package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

const removalTolerance = 0.000001

var errRemovalPreviewChanged = errors.New("出库单、库存或客户流水已变化，请重新预览后删除")

type removalSnapshot struct {
	Order        model.OutboundOrder
	Materials    []model.OutboundOrderMaterial
	Transactions []model.CustomerTransaction
	Customer     *model.Customer
	Inventories  []model.Inventory
}

type removalInventoryChange struct {
	Inventory model.Inventory
	Release   float64
	Restore   float64
}

type removalPlan struct {
	Snapshot       removalSnapshot
	Preview        types.OutboundRemovePreview
	TransactionIDs []primitive.ObjectID
	Changes        []removalInventoryChange
}

func removalError(message string) error {
	return &customerfinance.BusinessError{Message: message}
}

func removalTransactionOptions() *options.TransactionOptions {
	return options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority())
}

func baseRemovalPlan(snapshot removalSnapshot) *removalPlan {
	return &removalPlan{Snapshot: snapshot, Preview: types.OutboundRemovePreview{
		Id: snapshot.Order.Id.Hex(), Code: snapshot.Order.Code,
		CustomerName: snapshot.Order.CustomerName, Status: snapshot.Order.Status,
		Inventorys: make([]types.OutboundRemoveInventory, 0),
	}}
}

// Only these two transaction kinds belong to the outbound receivable. Payments,
// returns and manual adjustments are independent even when they mention its code.
func selectRemovalTransactions(order model.OutboundOrder, records []model.CustomerTransaction, uniqueCode bool) ([]model.CustomerTransaction, error) {
	selected := make([]model.CustomerTransaction, 0)
	for _, record := range records {
		typeCode := customerfinance.TransactionType(record)
		if typeCode != financeCode.TransactionTypeOutboundAR && typeCode != financeCode.TransactionTypeARAdjustment {
			continue
		}
		sourceID := strings.TrimSpace(record.SourceId)
		sourceType := strings.TrimSpace(record.SourceType)
		orderCode := strings.TrimSpace(record.OrderCode)
		sourceCode := strings.TrimSpace(record.SourceCode)
		codeMatches := orderCode == order.Code || sourceCode == order.Code
		if sourceID != order.Id.Hex() && !codeMatches {
			continue
		}
		// Explicit manual sources are independent customer entries; a referenced
		// order code does not turn them into this order's generated receivable.
		// An explicit ID claiming this order remains a source-identity conflict.
		independent, err := customerfinance.IndependentManualARAdjustment(record, order.Id.Hex())
		if err != nil {
			return nil, err
		}
		if independent {
			continue
		}
		// A code must never override an explicit, different source identity.
		if sourceID != "" && sourceID != order.Id.Hex() {
			return nil, removalError("存在单号相同但来源ID不同的应收流水，请先核对流水归属")
		}
		if order.CustomerId == "" || record.CustomerId != order.CustomerId {
			return nil, removalError("存在客户不匹配的应收流水，请先核对流水归属")
		}
		knownSource := sourceType == financeCode.SourceTypeOutboundOrder || sourceType == financeCode.SourceTypeSystemRecount
		if sourceID != "" && !knownSource || sourceID == "" && sourceType != "" && !knownSource {
			return nil, removalError("存在来源类型不匹配的应收流水，请先核对流水归属")
		}
		if orderCode != "" && orderCode != order.Code || sourceCode != "" && sourceCode != order.Code {
			return nil, removalError("存在来源单号不匹配的应收流水，请先核对流水归属")
		}
		if sourceID == "" && (orderCode != order.Code || !uniqueCode) {
			return nil, removalError("历史应收流水无法唯一归属到此出库单")
		}
		if record.Id.IsZero() {
			return nil, removalError("应收流水缺少有效ID")
		}
		selected = append(selected, record)
	}
	if err := validateRemovalAdjustmentLinks(selected); err != nil {
		return nil, err
	}
	return selected, nil
}

func validateRemovalAdjustmentLinks(records []model.CustomerTransaction) error {
	byID := make(map[primitive.ObjectID]model.CustomerTransaction, len(records))
	for _, record := range records {
		byID[record.Id] = record
	}
	for _, record := range records {
		if customerfinance.TransactionType(record) != financeCode.TransactionTypeARAdjustment || strings.TrimSpace(record.OriginalTransactionId) == "" {
			continue
		}
		seen := map[primitive.ObjectID]bool{record.Id: true}
		current := record
		for {
			parentID, err := primitive.ObjectIDFromHex(strings.TrimSpace(current.OriginalTransactionId))
			if err != nil {
				return removalError("应收调整的原流水关联ID异常，请先核对")
			}
			if seen[parentID] {
				return removalError("应收调整的原流水关联存在循环，请先核对")
			}
			parent, ok := byID[parentID]
			if !ok {
				return removalError("应收调整关联了缺失或不属于本单的原流水，请先核对")
			}
			if customerfinance.TransactionType(parent) == financeCode.TransactionTypeOutboundAR {
				break
			}
			seen[parentID] = true
			current = parent
		}
	}
	return nil
}

func removalInventoryChanges(order model.OutboundOrder, materials []model.OutboundOrderMaterial, inventories []model.Inventory) ([]removalInventoryChange, error) {
	mode := ""
	switch order.Status {
	case "预发货":
		mode = "none"
	case "待拣货":
		mode = "release"
	case "已拣货", "已打包", "已称重", "已出库", "已签收":
		mode = "restore"
	default:
		return nil, removalError("出库单状态不支持删除")
	}
	if mode != "none" && len(materials) == 0 {
		return nil, removalError("出库单缺少物料明细，无法核对库存恢复数量")
	}
	byID := make(map[primitive.ObjectID]model.Inventory, len(inventories))
	for _, inventory := range inventories {
		byID[inventory.Id] = inventory
	}
	quantities := make(map[primitive.ObjectID]decimal.Decimal)
	for _, material := range materials {
		if material.Id.IsZero() || material.OrderCode != order.Code {
			return nil, removalError("出库物料明细归属异常")
		}
		if mode == "none" {
			if len(material.Inventorys) > 0 {
				return nil, removalError("预发货出库单存在库存分配记录，请先核对库存状态")
			}
			continue
		}
		if !finitePositive(material.Quantity) || material.MaterialId == "" || strings.TrimSpace(material.Unit) == "" {
			return nil, removalError("出库物料数量、物料ID或单位异常")
		}
		if len(material.Inventorys) == 0 {
			return nil, removalError(fmt.Sprintf("物料[%s]缺少原库存分配记录，无法恢复", material.Name))
		}
		allocated := decimal.Zero
		for _, allocation := range material.Inventorys {
			id, err := primitive.ObjectIDFromHex(allocation.InventoryId)
			if err != nil || !finitePositive(allocation.ShipmentQuantity) {
				return nil, removalError("出库物料库存分配ID或数量异常")
			}
			inventory, ok := byID[id]
			if !ok {
				return nil, removalError(fmt.Sprintf("物料[%s]的原库存批次不存在，无法恢复", material.Name))
			}
			if inventory.MaterialId != material.MaterialId || inventory.Unit != material.Unit {
				return nil, removalError(fmt.Sprintf("物料[%s]的原库存批次物料或单位不匹配", material.Name))
			}
			if !finiteNonnegative(inventory.LockedQuantity) || !finiteNonnegative(inventory.AvailableQuantity) {
				return nil, removalError("原库存批次的锁定或可用数量异常")
			}
			quantity := decimal.NewFromFloat(allocation.ShipmentQuantity)
			allocated = allocated.Add(quantity)
			quantities[id] = quantities[id].Add(quantity)
		}
		if allocated.Sub(decimal.NewFromFloat(material.Quantity)).Abs().GreaterThan(decimal.NewFromFloat(removalTolerance)) {
			return nil, removalError(fmt.Sprintf("物料[%s]的库存分配数量与出库数量不一致", material.Name))
		}
	}
	changes := make([]removalInventoryChange, 0, len(quantities))
	for id, quantity := range quantities {
		inventory := byID[id]
		change := removalInventoryChange{Inventory: inventory}
		if mode == "release" {
			if decimal.NewFromFloat(inventory.LockedQuantity).LessThan(quantity) {
				return nil, removalError(fmt.Sprintf("物料[%s]原库存锁定数量不足，无法释放", inventory.Name))
			}
			change.Release = quantity.InexactFloat64()
		} else {
			change.Restore = quantity.InexactFloat64()
		}
		changes = append(changes, change)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Inventory.Id.Hex() < changes[j].Inventory.Id.Hex() })
	return changes, nil
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func prepareRemovalPlan(snapshot removalSnapshot, uniqueCode bool) (*removalPlan, error) {
	plan := baseRemovalPlan(snapshot)
	if !uniqueCode || strings.TrimSpace(snapshot.Order.Code) == "" {
		return plan, removalError("出库单号无法唯一定位明细及历史流水，请先核对")
	}
	selected, err := selectRemovalTransactions(snapshot.Order, snapshot.Transactions, uniqueCode)
	if err != nil {
		return plan, err
	}
	selectedIDs := make(map[primitive.ObjectID]bool, len(selected))
	removedNet := decimal.Zero
	for _, record := range selected {
		delta, e := customerfinance.TransactionNet(record)
		if e != nil {
			return plan, e
		}
		removedNet = removedNet.Add(delta)
		selectedIDs[record.Id] = true
		plan.TransactionIDs = append(plan.TransactionIDs, record.Id)
	}
	if removedNet.LessThan(decimal.NewFromFloat(-removalTolerance)) {
		return plan, removalError("本单应收流水净额异常，请先核对")
	}
	plan.Preview.TransactionCount = int64(len(selected))
	plan.Preview.ReceivableReduction = removedNet.InexactFloat64()
	if snapshot.Order.CustomerId != "" {
		if snapshot.Customer == nil || snapshot.Customer.Id.Hex() != snapshot.Order.CustomerId || snapshot.Customer.IsDeleted || snapshot.Customer.Status == "删除" {
			return plan, removalError("客户不存在或已删除，无法重建客户余额")
		}
		before, after := decimal.Zero, decimal.Zero
		for _, record := range snapshot.Transactions {
			if record.CustomerId != snapshot.Order.CustomerId {
				continue
			}
			delta, e := customerfinance.TransactionNet(record)
			if e != nil {
				return plan, e
			}
			before = before.Add(delta)
			if !selectedIDs[record.Id] {
				after = after.Add(delta)
			}
		}
		beforeAR, beforeCredit := customerfinance.SplitCustomerNetBalance(before)
		afterAR, afterCredit := customerfinance.SplitCustomerNetBalance(after)
		plan.Preview.ReceivableBefore, plan.Preview.CreditBefore = beforeAR.InexactFloat64(), beforeCredit.InexactFloat64()
		plan.Preview.ReceivableAfter, plan.Preview.CreditAfter = afterAR.InexactFloat64(), afterCredit.InexactFloat64()
	}
	plan.Changes, err = removalInventoryChanges(snapshot.Order, snapshot.Materials, snapshot.Inventories)
	if err != nil {
		return plan, err
	}
	for _, change := range plan.Changes {
		inventory := change.Inventory
		plan.Preview.Inventorys = append(plan.Preview.Inventorys, types.OutboundRemoveInventory{
			InventoryId: inventory.Id.Hex(), MaterialId: inventory.MaterialId, Name: inventory.Name,
			Model: inventory.Model, Unit: inventory.Unit, WarehouseName: inventory.WarehouseName,
			ReleaseLockedQuantity: change.Release, RestoreAvailableQuantity: change.Restore,
		})
	}
	plan.Preview.PreviewToken, err = removalPreviewToken(snapshot)
	if err != nil {
		return plan, removalError("单据、库存或客户流水含异常数据，无法生成删除预览")
	}
	plan.Preview.Allowed = true
	return plan, nil
}

// Canonicalize collection order, then bind the confirmation to every input used
// by the plan, including independent customer payments and current stock state.
func removalPreviewToken(snapshot removalSnapshot) (string, error) {
	snapshot.Materials = append([]model.OutboundOrderMaterial(nil), snapshot.Materials...)
	snapshot.Transactions = append([]model.CustomerTransaction(nil), snapshot.Transactions...)
	snapshot.Inventories = append([]model.Inventory(nil), snapshot.Inventories...)
	sort.Slice(snapshot.Materials, func(i, j int) bool { return snapshot.Materials[i].Id.Hex() < snapshot.Materials[j].Id.Hex() })
	sort.Slice(snapshot.Transactions, func(i, j int) bool { return snapshot.Transactions[i].Id.Hex() < snapshot.Transactions[j].Id.Hex() })
	sort.Slice(snapshot.Inventories, func(i, j int) bool { return snapshot.Inventories[i].Id.Hex() < snapshot.Inventories[j].Id.Hex() })
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(append([]byte("outbound-remove-v1:"), encoded...))
	return hex.EncodeToString(digest[:]), nil
}

func readRemovalPlan(ctx context.Context, svcCtx *svc.ServiceContext, id primitive.ObjectID) (*removalPlan, error) {
	var snapshot removalSnapshot
	if err := svcCtx.OutboundOrderModel.FindOne(ctx, bson.M{"_id": id}).Decode(&snapshot.Order); err != nil {
		return nil, err
	}
	plan := baseRemovalPlan(snapshot)
	orderCount, err := svcCtx.OutboundOrderModel.CountDocuments(ctx, bson.M{"code": snapshot.Order.Code})
	if err != nil {
		return plan, err
	}
	if err = readRemovalDocuments(ctx, svcCtx.OutboundMaterialModel, bson.M{"order_code": snapshot.Order.Code}, &snapshot.Materials); err != nil {
		return plan, err
	}
	conditions := bson.A{bson.M{"source_id": id.Hex()}, bson.M{"order_code": snapshot.Order.Code}, bson.M{"source_code": snapshot.Order.Code}}
	if snapshot.Order.CustomerId != "" {
		conditions = append(conditions, bson.M{"customer_id": snapshot.Order.CustomerId})
		customerID, e := primitive.ObjectIDFromHex(snapshot.Order.CustomerId)
		if e != nil {
			return plan, removalError("出库单客户ID异常")
		}
		var customer model.Customer
		e = svcCtx.CustomerModel.FindOne(ctx, bson.M{"_id": customerID}).Decode(&customer)
		if e != nil && e != mongo.ErrNoDocuments {
			return plan, e
		}
		if e == nil {
			snapshot.Customer = &customer
		}
	}
	if err = readRemovalDocuments(ctx, svcCtx.CustomerTransactionModel, bson.M{"$or": conditions}, &snapshot.Transactions); err != nil {
		return plan, err
	}
	inventoryIDs := make([]primitive.ObjectID, 0)
	seen := make(map[primitive.ObjectID]bool)
	for _, material := range snapshot.Materials {
		for _, allocation := range material.Inventorys {
			inventoryID, e := primitive.ObjectIDFromHex(allocation.InventoryId)
			if e == nil && !seen[inventoryID] {
				inventoryIDs = append(inventoryIDs, inventoryID)
				seen[inventoryID] = true
			}
		}
	}
	if len(inventoryIDs) > 0 {
		if err = readRemovalDocuments(ctx, svcCtx.InventoryModel, bson.M{"_id": bson.M{"$in": inventoryIDs}}, &snapshot.Inventories); err != nil {
			return plan, err
		}
	}
	return prepareRemovalPlan(snapshot, orderCount == 1)
}

func readRemovalDocuments(ctx context.Context, collection *mongo.Collection, filter interface{}, destination interface{}) error {
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, destination)
}

func removalOrderFilter(order model.OutboundOrder) bson.M {
	filter := outboundRevisionFilter(order.Id, order.RevisionNo)
	filter["status"] = order.Status
	return filter
}

// The conditional order delete is the first write and gates every side effect.
// All callers must pass the same MongoDB transaction context through this chain.
func applyRemovalPlan(ctx context.Context, svcCtx *svc.ServiceContext, plan *removalPlan) error {
	order := plan.Snapshot.Order
	result, err := svcCtx.OutboundOrderModel.DeleteOne(ctx, removalOrderFilter(order))
	if err != nil {
		return err
	}
	if result.DeletedCount != 1 {
		return removalError("出库单状态已变化，请刷新删除预览")
	}
	for _, change := range plan.Changes {
		inventory := change.Inventory
		filter := bson.M{"_id": inventory.Id, "material_id": inventory.MaterialId, "unit": inventory.Unit}
		if inventory.WarehouseId != "" {
			filter["warehouse_id"] = inventory.WarehouseId
		}
		inc := bson.M{}
		if change.Release > 0 {
			filter["locked_quantity"] = bson.M{"$gte": change.Release}
			inc["locked_quantity"] = -change.Release
		}
		if change.Restore > 0 {
			inc["available_quantity"] = change.Restore
		}
		updated, e := svcCtx.InventoryModel.UpdateOne(ctx, filter, bson.M{"$inc": inc})
		if e != nil {
			return e
		}
		if updated.MatchedCount != 1 || updated.ModifiedCount != 1 {
			return removalError("库存状态已变化，请刷新删除预览")
		}
	}
	materialIDs := make([]primitive.ObjectID, 0, len(plan.Snapshot.Materials))
	for _, material := range plan.Snapshot.Materials {
		materialIDs = append(materialIDs, material.Id)
	}
	if len(materialIDs) > 0 {
		result, err = svcCtx.OutboundMaterialModel.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": materialIDs}, "order_code": order.Code})
		if err != nil {
			return err
		}
		if result.DeletedCount != int64(len(materialIDs)) {
			return removalError("出库物料明细已变化，请刷新删除预览")
		}
	}
	if len(plan.TransactionIDs) > 0 {
		result, err = svcCtx.CustomerTransactionModel.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": plan.TransactionIDs}, "customer_id": order.CustomerId})
		if err != nil {
			return err
		}
		if result.DeletedCount != int64(len(plan.TransactionIDs)) {
			return removalError("客户流水已变化，请刷新删除预览")
		}
	}
	if order.CustomerId != "" {
		if err = rebuildRemovalCustomerBalance(ctx, svcCtx, order.CustomerId); err != nil {
			return err
		}
	}
	return materialdelivery.RebuildAfterOutboundRemoval(ctx, svcCtx, order, plan.Snapshot.Materials)
}

func rebuildRemovalCustomerBalance(ctx context.Context, svcCtx *svc.ServiceContext, customerID string) error {
	id, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return removalError("客户ID异常")
	}
	return customerfinance.RebuildCustomerBalance(ctx, svcCtx, id)
}
