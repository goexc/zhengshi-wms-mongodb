package receipt

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.InboundReceiptAddRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	var receipt model.InboundReceipt
	//receipt.Status = code.InboundReceiptStatusCode("待审核")
	receipt.Status = "待审核"
	receipt.Type = strings.TrimSpace(req.Type)
	receipt.Code = strings.TrimSpace(req.Code)
	receipt.ReceivingDate = req.ReceivingDate
	receipt.Remark = strings.TrimSpace(req.Remark)
	receipt.Annex = req.Annex

	//1.入库单号是否冲突
	var filter = bson.M{"code": req.Code, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.InboundReceiptModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询入库单[%s]是否冲突:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单号已占用"
		return resp, nil
	}

	//2.供应商是否存在
	if req.Type == "采购入库" || req.Type == "外协入库" {
		var supplier model.Supplier
		supplierId, _ := primitive.ObjectIDFromHex(req.SupplierId)
		singleRes := l.svcCtx.SupplierModel.FindOne(l.ctx, bson.M{"_id": supplierId})
		switch singleRes.Err() {
		case nil: //供应商存在
			if e := singleRes.Decode(&supplier); e != nil {
				fmt.Printf("[Error]解析供应商[%s]:%s\n", req.SupplierId, e.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //供应商不存在
			fmt.Printf("[Error]供应商[%s]不存在\n", req.SupplierId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "供应商不存在"
			return resp, nil
		default: //其他错误
			fmt.Printf("[Error]查询供应商[%s]是否存在:%s\n", req.SupplierId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		receipt.SupplierId = supplier.Id.Hex()
		receipt.SupplierName = supplier.Name
	}

	//3.客户是否存在
	if req.Type == "退货入库" {
		var customer model.Customer
		customerId, _ := primitive.ObjectIDFromHex(req.CustomerId)
		singleRes := l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId})
		switch singleRes.Err() {
		case nil: //客户存在
			if e := singleRes.Decode(&customer); e != nil {
				fmt.Printf("[Error]解析客户[%s]:%s\n", req.CustomerId, e.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //客户不存在
			fmt.Printf("[Error]客户[%s]不存在\n", req.CustomerId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户不存在"
			return resp, nil
		default: //其他错误
			fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.CustomerId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		receipt.CustomerId = customer.Id.Hex()
		receipt.CustomerName = customer.Name
	}
	//4.仓库、库区、货架、货位是否存在
	var materialsId = make([]primitive.ObjectID, 0)
	var warehousesId = make([]primitive.ObjectID, 0)
	var warehouseZonesId = make([]primitive.ObjectID, 0)
	var warehouseRacksId = make([]primitive.ObjectID, 0)
	var warehouseBinsId = make([]primitive.ObjectID, 0)

	var warehouses = make(map[string]model.Warehouse)         //仓库id=>“”
	var warehouseZones = make(map[string]model.WarehouseZone) //库区id=>“”
	var warehouseRacks = make(map[string]model.WarehouseRack) //货架id=>“”
	var warehouseBins = make(map[string]model.WarehouseBin)   //货位id=>“”

	//物料仓储信息
	var warehousing = make(map[string]types.Warehousing)

	//4.1 收集id
	for _, one := range req.Materials {
		materialId, _ := primitive.ObjectIDFromHex(strings.TrimSpace(one.Id))
		materialsId = append(materialsId, materialId)

		var warehouseId, zoneId, rackId, binId string
		if len(one.Position) >= 1 {
			warehouseId = one.Position[0]
			warehouseObjectID, _ := primitive.ObjectIDFromHex(one.Position[0]) //warehouseId已通过参数校验，无需再次判断
			warehousesId = append(warehousesId, warehouseObjectID)
		}
		if len(one.Position) >= 2 {
			zoneId = one.Position[1]
			zoneObjectID, _ := primitive.ObjectIDFromHex(one.Position[1]) //warehouseId已通过参数校验，无需再次判断
			warehouseZonesId = append(warehouseZonesId, zoneObjectID)
		}

		if len(one.Position) >= 3 {
			rackId = one.Position[2]
			rackObjectID, _ := primitive.ObjectIDFromHex(one.Position[2]) //warehouseId已通过参数校验，无需再次判断
			warehouseRacksId = append(warehouseRacksId, rackObjectID)
		}

		if len(one.Position) >= 4 {
			binId = one.Position[3]
			binObjectID, _ := primitive.ObjectIDFromHex(one.Position[3]) //warehouseId已通过参数校验，无需再次判断
			warehouseBinsId = append(warehouseBinsId, binObjectID)
		}

		warehousing[one.Id] = types.Warehousing{
			WarehouseId:     warehouseId,
			WarehouseZoneId: zoneId,
			WarehouseRackId: rackId,
			WarehouseBinId:  binId,
		}
	}

	//4.2 查询物料
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
	}

	var materialsMap = make(map[string]model.Material)
	for _, one := range ms {
		materialsMap[one.Id.Hex()] = one
	}

	var ws = make([]model.Warehouse, 0)
	var zs = make([]model.WarehouseZone, 0)
	var rs = make([]model.WarehouseRack, 0)
	var bs = make([]model.WarehouseBin, 0)

	//4.3.1 查询仓库
	if len(warehousesId) > 0 {
		cur, err = l.svcCtx.WarehouseModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": warehousesId}, "status": bson.M{"$ne": code.WarehouseStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询仓库列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
		defer cur.Close(l.ctx)

		if err = cur.All(l.ctx, &ws); err != nil {
			fmt.Printf("[Error]解析仓库列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		for _, one := range ws {
			warehouses[one.Id.Hex()] = one
		}
	}

	//4.3.2 仓库是否存在
	if len(warehousesId) > len(warehouses) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分仓库不存在"
		return resp, nil
	}

	for _, id := range warehousesId {
		if one, ok := warehouses[id.Hex()]; ok {
			if one.Status != code.WarehouseStatusCode("激活") {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("仓库[%s]%s，请选择其他仓库。", one.Name, code.WarehouseStatusText(one.Status))
				return resp, nil
			}
		}
	}

	//4.4.1 查询库区
	if len(warehouseZonesId) > 0 {
		cur, err = l.svcCtx.WarehouseZoneModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": warehouseZonesId}, "status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询库区列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
		defer cur.Close(l.ctx)

		if err = cur.All(l.ctx, &zs); err != nil {
			fmt.Printf("[Error]解析库区列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		for _, one := range zs {
			warehouseZones[one.Id.Hex()] = one
		}
	}
	//4.4.2 库区是否存在
	if len(warehouseZonesId) > len(warehouseZones) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分库区不存在"
		return resp, nil
	}

	for _, id := range warehouseZonesId {
		if one, ok := warehouseZones[id.Hex()]; ok {
			if one.Status != code.WarehouseZoneStatusCode("激活") {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("库区[%s]%s，请选择其他库区。", one.Name, code.WarehouseZoneStatusText(one.Status))
				return resp, nil
			}
		}
	}

	//4.5 查询货架
	if len(warehouseRacksId) > 0 {
		cur, err = l.svcCtx.WarehouseRackModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": warehouseRacksId}, "status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询货架列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
		defer cur.Close(l.ctx)

		if err = cur.All(l.ctx, &rs); err != nil {
			fmt.Printf("[Error]解析货架列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		for _, one := range rs {
			warehouseRacks[one.Id.Hex()] = one
		}
	}
	//4.5.2 货架是否存在
	if len(warehouseRacksId) > len(warehouseRacks) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分货架不存在"
		return resp, nil
	}

	for _, id := range warehouseRacksId {
		if one, ok := warehouseRacks[id.Hex()]; ok {
			if one.Status != code.WarehouseRackStatusCode("激活") {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("货架[%s]%s，请选择其他货架。", one.Name, code.WarehouseRackStatusText(one.Status))
				return resp, nil
			}
		}
	}

	//4.6 查询货位
	if len(warehouseBinsId) > 0 {
		cur, err = l.svcCtx.WarehouseBinModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": warehouseBinsId}, "status": bson.M{"$ne": code.WarehouseBinStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询货位列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
		defer cur.Close(l.ctx)

		if err = cur.All(l.ctx, &bs); err != nil {
			fmt.Printf("[Error]解析货位列表:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		for _, one := range bs {
			warehouseBins[one.Id.Hex()] = one
		}
	}
	//4.6.2 货位是否存在
	if len(warehouseBinsId) > len(warehouseBins) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分货位不存在"
		return resp, nil
	}

	for _, id := range warehouseBinsId {
		if one, ok := warehouseBins[id.Hex()]; ok {
			if one.Status != code.WarehouseBinStatusCode("激活") {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("货位[%s]%s，请选择其他货位。", one.Name, code.WarehouseBinStatusText(one.Status))
				return resp, nil
			}
		}
	}

	//5.仓库、库区、货架、货位是否存在
	var materials = make([]model.InboundMaterial, 0) //出库单的物料列表
	for _, one := range req.Materials {
		//累计金额
		receipt.TotalAmount += one.EstimatedQuantity * one.Price

		//收集物料列表
		im := model.InboundMaterial{
			Id:                one.Id,
			Index:             one.Index,
			Price:             one.Price,
			Name:              materialsMap[one.Id].Name,
			Model:             materialsMap[one.Id].Model,
			EstimatedQuantity: one.EstimatedQuantity,
			ActualQuantity:    0,
			Unit:              materialsMap[one.Id].Unit,
			//Status:            code.InboundReceiptStatusCode("未发货"),
			Status:            "未发货",
			WarehouseId:       warehousing[one.Id].WarehouseId,
			WarehouseZoneId:   warehousing[one.Id].WarehouseZoneId,
			WarehouseRackId:   warehousing[one.Id].WarehouseRackId,
			WarehouseBinId:    warehousing[one.Id].WarehouseBinId,
			WarehouseName:     warehouses[warehousing[one.Id].WarehouseId].Name,
			WarehouseZoneName: warehouseZones[warehousing[one.Id].WarehouseZoneId].Name,
			WarehouseRackName: warehouseRacks[warehousing[one.Id].WarehouseRackId].Name,
			WarehouseBinName:  warehouseBins[warehousing[one.Id].WarehouseBinId].Name,
		}

		//校验仓库、库区、货架、货位
		materials = append(materials, im)

		//收集物料价格
		if one.Price > 0 {
			var update = bson.M{
				"$set": bson.M{
					"material":   one.Id,
					"price":      one.Price,
					"created_at": time.Now().Unix(),
				},
			}

			//记录物料单价
			opts := options.Update().SetUpsert(true) //更新时，不存在就插入
			_, err = l.svcCtx.MaterialPriceModel.UpdateMany(l.ctx, bson.M{"material": one.Id, "price": one.Price}, update, opts)
			if err != nil {
				fmt.Printf("[Error]记录物料价格:%s\n", err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		}

	}

	//如果请求参数中的总金额>0，那么使用请求参数中的总金额
	if req.TotalAmount != 0 {
		receipt.TotalAmount = req.TotalAmount
	}

	if req.Type == "退货入库" && receipt.TotalAmount > 0 {
		receipt.TotalAmount *= -1
	}

	receipt.Materials = materials
	receipt.CreatorId = l.ctx.Value("uid").(string)
	receipt.CreatorName = l.ctx.Value("name").(string)
	receipt.CreatedAt = time.Now().Unix()
	receipt.UpdatedAt = time.Now().Unix()

	_, err = l.svcCtx.InboundReceiptModel.InsertOne(l.ctx, &receipt)
	if err != nil {
		fmt.Printf("[Error]新增入库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
