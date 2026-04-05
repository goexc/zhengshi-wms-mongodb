package outbound

import (
	"api/model"
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FastDepartureLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 极速出库
func NewFastDepartureLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FastDepartureLogic {
	return &FastDepartureLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FastDepartureLogic) FastDeparture(req *types.FastOutboundRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	now := time.Now().Unix()
	uid := l.ctx.Value("uid").(string)
	name := l.ctx.Value("name").(string)

	// 1. 出库单号是否冲突
	count, err := l.svcCtx.OutboundOrderModel.CountDocuments(l.ctx, bson.M{"code": req.Code, "status": bson.M{"$ne": "删除"}})
	if err != nil {
		fmt.Printf("[Error]查询出库单是否冲突:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单号已占用"
		return resp, nil
	}

	// 2. 客户是否存在
	var customer model.Customer
	customerId, _ := primitive.ObjectIDFromHex(req.CustomerId)
	err = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId}).Decode(&customer)
	if err != nil {
		fmt.Printf("[Error]未找到客户:%s\n", err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	// 3. 收集物料信息
	var materialIds []primitive.ObjectID
	for _, one := range req.Materials {
		mid, _ := primitive.ObjectIDFromHex(strings.TrimSpace(one.MaterialId))
		materialIds = append(materialIds, mid)
	}

	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": materialIds}})
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询物料信息失败"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Material
	cur.All(l.ctx, &ms)
	var materialsMap = make(map[string]model.Material)
	for _, m := range ms {
		materialsMap[m.Id.Hex()] = m
	}

	if len(materialsMap) != len(req.Materials) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分物料不存在"
		return resp, nil
	}

	// 4. 查询并聚合库存
	var inventories []model.Inventory
	invCur, err := l.svcCtx.InventoryModel.Find(l.ctx, bson.M{"material_id": bson.M{"$in": materialIds}})
	if err == nil {
		invCur.All(l.ctx, &inventories)
		invCur.Close(l.ctx)
	}

	inventoryMap := make(map[string]float64)
	for _, inv := range inventories {
		inventoryMap[inv.MaterialId] += inv.AvailableQuantity
	}

	var inboundMaterials []model.InboundMaterial
	var inboundInventories []interface{}

	// 5. 检查库存并生成缺口入库单
	for index, mReq := range req.Materials {
		material := materialsMap[mReq.MaterialId]

		available := inventoryMap[mReq.MaterialId]
		if available < mReq.Quantity {
			shortfall := mReq.Quantity - available
			inboundQty := math.Ceil(shortfall/10000.0) * 10000

			// 自动入库单明细 (价格强制为0)
			im := model.InboundMaterial{
				Id:                material.Id.Hex(),
				Index:             index + 1,
				Price:             0,
				Name:              material.Name,
				Model:             material.Model,
				EstimatedQuantity: inboundQty,
				ActualQuantity:    inboundQty,
				Unit:              material.Unit,
				Status:            "入库完成",
			}
			inboundMaterials = append(inboundMaterials, im)

			// 在内存中直接生成库存记录
			newInv := model.Inventory{
				Type:              "生产入库",
				ReceiptCode:       fmt.Sprintf("AUTO_IN_%s_%d", req.Code, index),
				ReceiveCode:       fmt.Sprintf("BATCH_%s_%d", req.Code, index),
				EntryTime:         now,
				MaterialId:        material.Id.Hex(),
				Name:              material.Name,
				Price:             0,
				Model:             material.Model,
				Unit:              material.Unit,
				Quantity:          inboundQty,
				AvailableQuantity: inboundQty,
				CreatorId:         uid,
				CreatorName:       name,
				CreatedAt:         now,
			}
			inboundInventories = append(inboundInventories, newInv)
			// 更新可扣减内存库存
			inventories = append(inventories, newInv)
		}
	}

	// 6. 执行自动入库
	if len(inboundMaterials) > 0 {
		inboundReceipt := model.InboundReceipt{
			Status:        "入库完成",
			Type:          "生产入库", // 根据要求设置
			Code:          fmt.Sprintf("AUTO_IN_%s", req.Code),
			ReceivingDate: now,
			TotalAmount:   0,
			Materials:     inboundMaterials,
			CreatorId:     uid,
			CreatorName:   name,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		_, err = l.svcCtx.InboundReceiptModel.InsertOne(l.ctx, &inboundReceipt)
		if err != nil {
			resp.Code = http.StatusInternalServerError
			resp.Msg = "自动生成入库单失败"
			return resp, nil
		}
		
		if len(inboundInventories) > 0 {
			_, err = l.svcCtx.InventoryModel.InsertMany(l.ctx, inboundInventories)
			if err != nil {
				resp.Code = http.StatusInternalServerError
				resp.Msg = "增加库存失败"
				return resp, nil
			}
			// 增加完毕后重新拉取这批带新生成_id的记录（以备后续出库扣减匹配）
			invCur, _ := l.svcCtx.InventoryModel.Find(l.ctx, bson.M{"material_id": bson.M{"$in": materialIds}})
			inventories = nil
			invCur.All(l.ctx, &inventories)
			invCur.Close(l.ctx)
		}
	}

	// 7. 处理出库逻辑
	status := "预发货"
	if req.ReceiptTime > 0 {
		status = "已签收"
	} else if req.DepartureTime > 0 {
		status = "已出库"
	} else if req.WeighingTime > 0 {
		status = "已称重"
	} else if req.PackingTime > 0 {
		status = "已打包"
	} else if req.PickingTime > 0 {
		status = "已拣货"
	}

	isPack := 0
	if req.PackingTime > 0 {
		isPack = 1
	}
	isWeigh := 0
	if req.WeighingTime > 0 {
		isWeigh = 1
	}

	outboundOrder := model.OutboundOrder{
		Status:        status,
		IsPack:        isPack,
		IsWeigh:       isWeigh,
		Type:          req.Type,
		Code:          req.Code,
		CustomerId:    customer.Id.Hex(),
		CustomerName:  customer.Name,
		TotalAmount:   0, // 出库单总金额为0
		CreatorId:     uid,
		CreatorName:   name,
		ConfirmTime:   now,
		PickingTime:   req.PickingTime,
		PackingTime:   req.PackingTime,
		WeighingTime:  req.WeighingTime,
		DepartureTime: req.DepartureTime,
		ReceiptTime:   req.ReceiptTime,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	var outboundMaterials []interface{}
	for index, mReq := range req.Materials {
		material := materialsMap[mReq.MaterialId]

		// 扣减库存
		qtyToDeduct := mReq.Quantity
		for i, inv := range inventories {
			if inv.MaterialId == mReq.MaterialId && inv.AvailableQuantity > 0 {
				take := math.Min(qtyToDeduct, inv.AvailableQuantity)
				inv.AvailableQuantity -= take
				inv.Quantity -= take
				qtyToDeduct -= take
				inventories[i] = inv

				update := bson.M{"$set": bson.M{"available_quantity": inv.AvailableQuantity, "quantity": inv.Quantity}}
				_, updateErr := l.svcCtx.InventoryModel.UpdateOne(l.ctx, bson.M{"_id": inv.Id}, update)
				if updateErr != nil {
					logx.Errorf("Update inventory error: %v", updateErr)
				}

				if qtyToDeduct <= 0 {
					break
				}
			}
		}

		om := model.OutboundOrderMaterial{
			OrderCode:  req.Code,
			MaterialId: material.Id.Hex(),
			Index:      index + 1,
			Price:      0, // 出库单物料价格为0
			Name:       material.Name,
			Model:      material.Model,
			Quantity:   mReq.Quantity,
			Unit:       material.Unit,
		}
		outboundMaterials = append(outboundMaterials, om)
	}

	_, err = l.svcCtx.OutboundOrderModel.InsertOne(l.ctx, &outboundOrder)
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "新增出库单失败"
		return resp, nil
	}

	if len(outboundMaterials) > 0 {
		_, err = l.svcCtx.OutboundMaterialModel.InsertMany(l.ctx, outboundMaterials)
	}

	resp.Code = http.StatusOK
	resp.Msg = "极速出库成功"
	return resp, nil
}
