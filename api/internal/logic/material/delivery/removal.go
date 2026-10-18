package delivery

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"api/internal/svc"
	"api/model"
	quoteCode "api/pkg/code"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const sourceInvalidReasonOutboundRemoved = "source_outbound_order_removed"

// HasOutboundQuoteReference reserves a deleted order number while any quote still
// records it as historical evidence, including void or invalid quotes.
func HasOutboundQuoteReference(ctx context.Context, svcCtx *svc.ServiceContext, orderCode string) (bool, error) {
	err := svcCtx.MaterialQuoteModel.FindOne(ctx, bson.M{"source_order_code": strings.TrimSpace(orderCode)},
		options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	return err == nil, err
}

// RebuildAfterOutboundRemoval runs in the caller's deletion transaction, after
// the order and its lines have been removed. It only changes affected projections;
// quotations retain their original source order instead of being reattached.
func RebuildAfterOutboundRemoval(ctx context.Context, svcCtx *svc.ServiceContext, order model.OutboundOrder, materials []model.OutboundOrderMaterial) error {
	if mongo.SessionFromContext(ctx) == nil {
		return fmt.Errorf("删除出库单后重建交付记录必须在删除事务内执行")
	}
	ids, err := deliveryMaterialIDs(materials)
	if err != nil {
		return err
	}
	if err = lockDeliveryMaterials(ctx, svcCtx, ids); err != nil {
		return err
	}
	now := time.Now().Unix()
	invalidQuotes, err := invalidateRemovedOutboundQuotes(ctx, svcCtx, order, ids, now)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	summaries, globals, err := remainingMaterialDeliveries(ctx, svcCtx, ids, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		key := customerMaterialDeliveryKey(order.CustomerId, id.Hex())
		if order.CustomerId != "" {
			if summary, ok := summaries[key]; ok {
				if err = resetRemovedQuoteSummary(ctx, svcCtx, order.CustomerId, id.Hex(), order.Code, invalidQuotes, now); err != nil {
					return err
				}
				if err = upsertDeliverySummary(ctx, svcCtx, summary.record, now); err != nil {
					return err
				}
			} else if err = cleanupStaleCustomerMaterialDeliveries(ctx, svcCtx, nil, now, false,
				bson.M{"customer_id": order.CustomerId, "material_id": id.Hex()}); err != nil {
				return err
			}
		}
		if err = updateGlobalFirstDelivery(ctx, svcCtx, id, globals[id.Hex()], now); err != nil {
			return err
		}
	}
	return nil
}

func deliveryMaterialIDs(materials []model.OutboundOrderMaterial) ([]primitive.ObjectID, error) {
	unique := make(map[string]primitive.ObjectID, len(materials))
	for _, material := range materials {
		id, err := primitive.ObjectIDFromHex(material.MaterialId)
		if err != nil {
			return nil, fmt.Errorf("出库物料id[%s]格式错误:%w", material.MaterialId, err)
		}
		unique[id.Hex()] = id
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ids := make([]primitive.ObjectID, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, unique[key])
	}
	return ids, nil
}

// The revision write participates in the existing business transaction. It
// creates a write conflict with another departure/removal or a rebuild publisher
// for the same material, including when no customer projection currently exists.
func lockDeliveryMaterials(ctx context.Context, svcCtx *svc.ServiceContext, ids []primitive.ObjectID) error {
	for _, id := range ids {
		result, err := svcCtx.MaterialModel.UpdateByID(ctx, id, bson.M{"$inc": bson.M{"delivery_projection_revision": int64(1)}})
		if err != nil {
			return fmt.Errorf("锁定物料[%s]交付投影失败:%w", id.Hex(), err)
		}
		if result.MatchedCount != 1 {
			return fmt.Errorf("物料[%s]不存在，无法重建交付记录", id.Hex())
		}
	}
	return nil
}

func remainingMaterialDeliveries(ctx context.Context, svcCtx *svc.ServiceContext, ids []primitive.ObjectID, now int64) (map[string]rebuildSummary, map[string]materialFirstDelivery, error) {
	materialIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		materialIDs = append(materialIDs, id.Hex())
	}
	cur, err := svcCtx.OutboundMaterialModel.Find(ctx, bson.M{"material_id": bson.M{"$in": materialIDs}})
	if err != nil {
		return nil, nil, fmt.Errorf("查询剩余交付明细失败:%w", err)
	}
	defer cur.Close(ctx)
	var materials []model.OutboundOrderMaterial
	if err = cur.All(ctx, &materials); err != nil {
		return nil, nil, fmt.Errorf("解析剩余交付明细失败:%w", err)
	}
	byOrder := groupDeliveryMaterials(materials)
	orderCodes := make([]string, 0, len(byOrder))
	for code := range byOrder {
		orderCodes = append(orderCodes, code)
	}
	summaries := make(map[string]rebuildSummary)
	globals := make(map[string]materialFirstDelivery)
	if len(orderCodes) == 0 {
		return summaries, globals, nil
	}
	filter := customerDeliveryOrderFilter()
	filter["code"] = bson.M{"$in": orderCodes}
	orderCur, err := svcCtx.OutboundOrderModel.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "departure_time", Value: 1}, {Key: "code", Value: 1}}))
	if err != nil {
		return nil, nil, fmt.Errorf("查询剩余交付出库单失败:%w", err)
	}
	defer orderCur.Close(ctx)
	for orderCur.Next(ctx) {
		var order model.OutboundOrder
		if err = orderCur.Decode(&order); err != nil {
			return nil, nil, err
		}
		if err = accumulateDeliveryOrder(order, byOrder[order.Code], summaries, globals, now); err != nil {
			return nil, nil, err
		}
	}
	return summaries, globals, orderCur.Err()
}

func invalidateRemovedOutboundQuotes(ctx context.Context, svcCtx *svc.ServiceContext, order model.OutboundOrder, materialIDs []primitive.ObjectID, now int64) ([]string, error) {
	// Read the original first-delivery source before its projection is rebuilt.
	// Older prices can reference only that delivery, without a quote ID.
	materialSet := make(map[string]struct{}, len(materialIDs))
	materialTexts := make([]string, 0, len(materialIDs))
	for _, id := range materialIDs {
		materialSet[id.Hex()] = struct{}{}
		materialTexts = append(materialTexts, id.Hex())
	}
	deliveryCur, err := svcCtx.CustomerMaterialDeliveryModel.Find(ctx,
		bson.M{"customer_id": order.CustomerId, "material_id": bson.M{"$in": materialTexts}})
	if err != nil {
		return nil, fmt.Errorf("查询被删首次交付来源失败:%w", err)
	}
	defer deliveryCur.Close(ctx)
	var deliveries []model.CustomerMaterialDelivery
	if err = deliveryCur.All(ctx, &deliveries); err != nil {
		return nil, err
	}
	deliveryIDs := make([]string, 0, len(deliveries))
	ownedDeliveries := make(map[string]struct{}, len(deliveries))
	for _, delivery := range deliveries {
		ownedDeliveries[delivery.Id.Hex()] = struct{}{}
		if delivery.FirstDeliveryOrderCode == strings.TrimSpace(order.Code) {
			deliveryIDs = append(deliveryIDs, delivery.Id.Hex())
		}
	}
	filter := bson.M{"$or": []bson.M{
		{"source_order_id": order.Id.Hex()}, {"source_order_code": strings.TrimSpace(order.Code)},
	}}
	cur, err := svcCtx.MaterialQuoteModel.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("查询被删出库单关联报价失败:%w", err)
	}
	defer cur.Close(ctx)
	var quotes []model.MaterialQuote
	if err = cur.All(ctx, &quotes); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(quotes))
	objectIDs := make([]primitive.ObjectID, 0, len(quotes))
	for _, quote := range quotes {
		if quoteBelongsToRemovedOutbound(quote, order, materialSet, ownedDeliveries) {
			ids = append(ids, quote.Id.Hex())
			objectIDs = append(objectIDs, quote.Id)
		}
	}
	priceSources := make([]bson.M, 0, 2)
	if len(ids) > 0 {
		set := bson.M{"source_valid": false, "source_invalid_reason": sourceInvalidReasonOutboundRemoved, "updated_at": now}
		if _, err = svcCtx.MaterialQuoteModel.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": objectIDs}}, bson.M{"$set": set}); err != nil {
			return nil, fmt.Errorf("标记被删出库单报价来源失效失败:%w", err)
		}
		priceSources = append(priceSources, bson.M{"source_quote_id": bson.M{"$in": ids}})
	}
	if len(deliveryIDs) > 0 {
		priceSources = append(priceSources, bson.M{
			"source_delivery_id": bson.M{"$in": deliveryIDs},
			"source_quote_id":    bson.M{"$in": []interface{}{"", nil}},
		})
	}
	if len(priceSources) == 0 {
		return ids, nil
	}
	if _, err = svcCtx.MaterialPriceModel.UpdateMany(ctx, bson.M{"$or": priceSources}, bson.M{"$set": bson.M{
		"source_valid": false, "source_invalid_reason": sourceInvalidReasonOutboundRemoved,
	}}); err != nil {
		return nil, fmt.Errorf("标记被删出库单报价价格失效失败:%w", err)
	}
	return ids, nil
}

func quoteBelongsToRemovedOutbound(quote model.MaterialQuote, order model.OutboundOrder, materials, ownedDeliveries map[string]struct{}) bool {
	if id := strings.TrimSpace(quote.SourceOrderId); id != "" {
		return id == order.Id.Hex()
	}
	if strings.TrimSpace(quote.SourceOrderCode) != strings.TrimSpace(order.Code) {
		return false
	}
	customerID, materialID := strings.TrimSpace(quote.CustomerId), strings.TrimSpace(quote.MaterialId)
	if customerID != "" && customerID != order.CustomerId {
		return false
	}
	if materialID != "" {
		if _, ok := materials[materialID]; !ok {
			return false
		}
	}
	if deliveryID := strings.TrimSpace(quote.DeliveryId); deliveryID != "" {
		_, owned := ownedDeliveries[deliveryID]
		return owned
	}
	// A legacy quote without a delivery reference needs both explicit identities.
	return customerID != "" && materialID != ""
}

func resetRemovedQuoteSummary(ctx context.Context, svcCtx *svc.ServiceContext, customerID, materialID, orderCode string, quoteIDs []string, now int64) error {
	invalidSources := []bson.M{{
		"first_delivery_order_code": strings.TrimSpace(orderCode),
		"latest_quote_id":           bson.M{"$in": []interface{}{"", nil}},
	}}
	if len(quoteIDs) > 0 {
		invalidSources = append(invalidSources, bson.M{"latest_quote_id": bson.M{"$in": quoteIDs}})
	}
	_, err := svcCtx.CustomerMaterialDeliveryModel.UpdateOne(ctx, bson.M{
		"customer_id": customerID, "material_id": materialID, "$or": invalidSources,
	}, bson.M{"$set": bson.M{
		"quote_status": quoteCode.QuoteStatusUnquoted, "latest_quote_id": "", "latest_quote_no": "", "latest_price": float64(0), "updated_at": now,
	}})
	return err
}

func updateGlobalFirstDelivery(ctx context.Context, svcCtx *svc.ServiceContext, id primitive.ObjectID, first materialFirstDelivery, now int64) error {
	_, err := svcCtx.MaterialModel.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"first_delivery_time": first.time, "first_delivery_order_code": first.orderCode, "updated_at": now,
	}})
	if err != nil {
		return fmt.Errorf("更新物料[%s]全局首次交付失败:%w", id.Hex(), err)
	}
	return nil
}
