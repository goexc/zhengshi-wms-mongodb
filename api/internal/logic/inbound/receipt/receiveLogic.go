package receipt

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReceiveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveLogic {
	return &ReceiveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 可用的承运商状态
var carrierStatus = map[string]string{"审核通过": "", "活动": ""}

func (l *ReceiveLogic) Receive(req *types.InboundReceiptReceiveRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	var actualQuantity float64 //入库物料总数量
	var statusCount int        //未发货物料数量
	for _, one := range req.Materials {
		//参数校验：入库物料总数不能是0
		actualQuantity += one.ActualQuantity
		//参数校验：入库状态不能全部是“未发货”
		if one.Status == "未发货" {
			statusCount++
		}
	}

	if actualQuantity == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "没有入库物料"
		return resp, nil
	}

	if statusCount == len(req.Materials) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	}

	receiptId, _ := primitive.ObjectIDFromHex(req.Id)
	var receipt model.InboundReceipt

	//1.入库单号是否存在
	var filter = bson.M{"_id": receiptId}
	singleRes := l.svcCtx.InboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析入库单:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //入库单未占用
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询入库单:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//待审核、审核不通过：不能执行发货/入库操作
	//if receipt.Status == code.InboundReceiptStatusCode("待审核") || receipt.Status == code.InboundReceiptStatusCode("审核不通过") {
	if receipt.Status == "待审核" || receipt.Status == "审核不通过" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请审核通过再操作"
		return resp, nil
	}

	//入库完成：不能继续执行发货/入库操作
	//if receipt.Status == code.InboundReceiptStatusCode("入库完成") {
	if receipt.Status == "入库完成" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单已完成，不能操作发货/入库"
		return resp, nil
	}
	//从未入库的入库单，总金额置为0
	fmt.Printf("[Info]发货单状态:%s\n", receipt.Status)
	if receipt.Status == "审核通过" || receipt.Status == "未发货" {
		receipt.TotalAmount = 0
	}

	//2.入库单状态
	//var statuses = make(map[int]int, 0)
	var statuses = make(map[string]int, 0)

	//3.承运商是否存在
	var carrier model.Carrier
	if req.CarrierId != "" {
		carrierId, _ := primitive.ObjectIDFromHex(req.CarrierId)
		singleRes = l.svcCtx.CarrierModel.FindOne(l.ctx, bson.M{"_id": carrierId})
		switch singleRes.Err() {
		case nil:
			if err = singleRes.Decode(&carrier); err != nil {
				fmt.Printf("[Error]解析承运商[%s]:%s\n", req.CarrierId, err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}

			if _, ok := carrierStatus[carrier.Status]; !ok {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("[Error][%s]的承运商不可用", carrier.Status)
				return resp, nil
			}

		case mongo.ErrNoDocuments: //承运商不存在
			resp.Code = http.StatusBadRequest
			resp.Msg = "承运商不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]查询承运商[%s]:%s\n", req.CarrierId, singleRes.Err().Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	//3.入库单总金额累计运费、其他费用
	//receipt.TotalAmount += req.CarrierCost
	//receipt.TotalAmount += req.OtherCost

	//4.入库单物料列表
	var materials = make(map[string]model.InboundMaterial)
	for _, one := range receipt.Materials {
		materials[one.Id] = one
	}

	//3.更改物料状态、实际入库数量
	if len(receipt.Materials) != len(req.Materials) {
		fmt.Printf("[Error]入库单物料数量:[%d],表单物料数量:[%d]\n", len(receipt.Materials), len(req.Materials))
		resp.Code = http.StatusBadRequest
		resp.Msg = "物料数量不一致"
		return resp, nil
	}

	//var materialsId = make([]primitive.ObjectID, 0)
	var materialsMap = make(map[string]types.InboundReceived)
	//for idx, one := range req.Materials {
	for idx := range req.Materials {
		//materialId, _ := primitive.ObjectIDFromHex(one.Id)
		//materialsId = append(materialsId, materialId)

		//if req.Materials[idx].ActualQuantity == 0 {
		//	continue
		//}
		materialsMap[req.Materials[idx].Id] = req.Materials[idx]
	}

	/*//3.2 查询物料
	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": materialsId}})
	if err != nil {
		fmt.Printf("[Error]查询物料列表失败：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Material
	if err = cur.All(l.ctx, &ms); err != nil {
		fmt.Printf("[Error]解析物料分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if len(materialsId) != len(ms) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分物料不存在"
		return resp, nil
	}*/

	//var materials = make(map[string]model.Material)
	//for _, one := range ms {
	//	materials[one.Id.Hex()] = one
	//}

	//仓储信息
	var warehousing = make(map[string][]string)

	//批次入库总金额
	var totalAmount decimal.Decimal
	for idx, one := range receipt.Materials {
		if _, ok := materialsMap[one.Id]; !ok {
			fmt.Printf("[Error]物料[%s]缺少状态值\n", one.Name)
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("物料[%s]未设置入库状态", one.Name)
			return resp, nil
		}

		//receipt.Materials[idx].Status = code.InboundReceiptStatusCode(materialsMap[one.Id].Status)
		receipt.Materials[idx].Status = materialsMap[one.Id].Status
		receipt.Materials[idx].ActualQuantity += materialsMap[one.Id].ActualQuantity
		statuses[receipt.Materials[idx].Status]++

		//物料金额
		var amount decimal.Decimal
		//amount.Mul(big.NewFloat(materialsMap[one.Id].ActualQuantity), big.NewFloat(receipt.Materials[idx].Price)).Float64()
		amount = decimal.NewFromFloat(materialsMap[one.Id].ActualQuantity).Mul(decimal.NewFromFloat(receipt.Materials[idx].Price))

		//累计总金额
		if receipt.Type == "退货入库" {
			totalAmount = totalAmount.Sub(amount)
			receipt.TotalAmount = decimal.NewFromFloat(receipt.TotalAmount).Sub(amount).InexactFloat64()
		} else {
			totalAmount = totalAmount.Add(amount)
			receipt.TotalAmount = decimal.NewFromFloat(receipt.TotalAmount).Add(amount).InexactFloat64()
		}

		//收集仓储信息
		if materialsMap[one.Id].ActualQuantity > 0 { //未到货的物料无需处理仓储信息
			warehousing[one.Id] = materialsMap[one.Id].Position
		}
	}

	//4.收集仓库、库区、货架、货位id
	warehouses, warehouseZones, warehouseRacks, warehouseBins, err := Warehousing(l.ctx, l.svcCtx, warehousing)
	if err != nil {
		resp.Code = http.StatusBadRequest
		resp.Msg = err.Error()
		return resp, nil
	}

	var receiveMaterials = make([]model.InboundReceiveMaterial, 0) //批次入库的物料列表
	for _, one := range req.Materials {
		if one.ActualQuantity == 0 {
			continue
		}

		//收集物料列表
		//忽略未到货的物料
		if one.Status == "未到货" {
			continue
		}
		//忽略之前作废、入库完成的物料
		//if materials[one.Id].Status == code.InboundReceiptStatusCode("作废") || materials[one.Id].Status == code.InboundReceiptStatusCode("入库完成") {
		if materials[one.Id].Status == "作废" || materials[one.Id].Status == "入库完成" {
			continue
		}

		im := model.InboundReceiveMaterial{
			Id:             one.Id,
			Index:          one.Index,
			Price:          one.Price,
			Name:           materials[one.Id].Name,
			Model:          materials[one.Id].Model,
			Unit:           materials[one.Id].Unit,
			ActualQuantity: one.ActualQuantity,
			//Status:            code.InboundReceiptStatusCode(one.Status),
			Status: one.Status,
			//WarehouseId:       warehouses[one.Position[0]].Id.Hex(),
			//WarehouseName:     warehouses[one.Position[0]].Name,
			//WarehouseZoneId:   warehouseZones[one.Position[1]].Id.Hex(),
			//WarehouseZoneName: warehouseZones[one.Position[1]].Name,
			//WarehouseRackId:   warehouseRacks[one.Position[2]].Id.Hex(),
			//WarehouseRackName: warehouseRacks[one.Position[2]].Name,
			//WarehouseBinId:    warehouseBins[one.Position[3]].Id.Hex(),
			//WarehouseBinName:  warehouseBins[one.Position[3]].Name,
		}

		if len(one.Position) == 4 {
			im.WarehouseBinId = warehouseBins[one.Position[3]].Id.Hex()
			im.WarehouseBinName = warehouseBins[one.Position[3]].Name
		}
		if len(one.Position) >= 3 {
			im.WarehouseRackId = warehouseRacks[one.Position[2]].Id.Hex()
			im.WarehouseRackName = warehouseRacks[one.Position[2]].Name
		}

		if len(one.Position) >= 2 {
			im.WarehouseZoneId = warehouseZones[one.Position[1]].Id.Hex()
			im.WarehouseZoneName = warehouseZones[one.Position[1]].Name
		}

		if len(one.Position) >= 1 {
			im.WarehouseId = warehouses[one.Position[0]].Id.Hex()
			im.WarehouseName = warehouses[one.Position[0]].Name
		}

		receiveMaterials = append(receiveMaterials, im)
	}

	//批次入库
	var inboundReceive = model.InboundReceive{
		InboundReceiptId: req.Id,
		Code:             req.Code,
		CarrierId:        req.CarrierId,
		CarrierName:      carrier.Name,
		CarrierCost:      req.CarrierCost,
		OtherCost:        req.OtherCost,
		TotalAmount:      totalAmount.InexactFloat64(),
		ReceivingDate:    req.ReceivingDate,
		Materials:        receiveMaterials,
		Annex:            nil,
		Remark:           req.Remark,
		CreatorId:        l.ctx.Value("uid").(string),
		CreatorName:      l.ctx.Value("name").(string),
		CreatedAt:        time.Now().Unix(),
	}

	//4.记录批次入库
	//4.1 查询仓库
	_, err = l.svcCtx.InboundReceiptReceiveModel.InsertOne(l.ctx, &inboundReceive)
	if err != nil {
		fmt.Printf("[Error]批次入库:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.更新物料状态和入库单状态
	update := bson.M{
		"$set": bson.M{
			"status":       code.Material2InboundReceiptStatus(statuses),
			"total_amount": receipt.TotalAmount,
			"materials":    receipt.Materials,
		},
	}
	_, err = l.svcCtx.InboundReceiptModel.UpdateByID(l.ctx, receipt.Id, &update)
	if err != nil {
		fmt.Printf("[Error]更新入库单[%s]物料状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//TODO:6.添加流水、库存
	switch true {
	//6.1 采购入库、外协入库流水
	case receipt.Type == "采购入库", receipt.Type == "外协入库", receipt.Type == "生产入库":

	//6.2 退货入库流水
	case receipt.Type == "退货入库":

	default:
		fmt.Printf("[Error]入库类型[%s]异常，本批次入库未计入流水\n", receipt.Type)
		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("[Error]入库类型[%s]异常，本批次入库未计入流水", receipt.Type)
		return resp, nil
	}

	//6.3 添加库存
	var inventorys = make([]interface{}, 0)
	for _, one := range receiveMaterials {
		inventorys = append(inventorys, model.Inventory{
			Type:              receipt.Type,
			WarehouseId:       one.WarehouseId,
			WarehouseName:     one.WarehouseName,
			WarehouseZoneId:   one.WarehouseZoneId,
			WarehouseZoneName: one.WarehouseZoneName,
			WarehouseRackId:   one.WarehouseRackId,
			WarehouseRackName: one.WarehouseRackName,
			WarehouseBinId:    one.WarehouseBinId,
			WarehouseBinName:  one.WarehouseBinName,
			ReceiptCode:       receipt.Code,
			ReceiveCode:       req.Code,
			//ReceivingDate:     req.ReceivingDate,
			EntryTime:         time.Now().Unix(),
			MaterialId:        one.Id,
			Name:              one.Name,
			Model:             one.Model,
			Price:             one.Price,
			Unit:              one.Unit,
			Quantity:          one.ActualQuantity,
			AvailableQuantity: one.ActualQuantity,
			CreatorId:         l.ctx.Value("uid").(string),
			CreatorName:       l.ctx.Value("name").(string),
			CreatedAt:         time.Now().Unix(),
		})
	}

	_, err = l.svcCtx.InventoryModel.InsertMany(l.ctx, inventorys)
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
