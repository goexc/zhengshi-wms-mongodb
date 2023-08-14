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
	receipt.Status = code.InboundReceiptStatusCode("待审核")
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
	if req.Type != "退货入库" {
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
	var materials = make([]model.InboundMaterial, 0)                //入库单的物料列表
	var warehousesStatus = make(map[string]model.Warehouse)         //仓库id=>“”
	var warehouseZonesStatus = make(map[string]model.WarehouseZone) //库区id=>“”
	var warehouseRacksStatus = make(map[string]model.WarehouseRack) //货架id=>“”
	var warehouseBinsStatus = make(map[string]model.WarehouseBin)   //货位id=>“”
	for _, one := range req.Materials {
		materialId, e := primitive.ObjectIDFromHex(strings.TrimSpace(one.Id))
		if materialId.IsZero() {
			fmt.Printf("[Error]解析物料id[%s]为空：%s\n", one.Id, e.Error())
			resp.Msg = "物料参数错误"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}

		warehouseId, _ := primitive.ObjectIDFromHex(one.WarehouseId) //warehouseId已通过参数校验，无需再次判断
		zoneId, _ := primitive.ObjectIDFromHex(one.WarehouseZoneId)  //warehouseId已通过参数校验，无需再次判断
		rackId, _ := primitive.ObjectIDFromHex(one.WarehouseRackId)  //warehouseId已通过参数校验，无需再次判断
		binId, _ := primitive.ObjectIDFromHex(one.WarehouseBinId)    //warehouseId已通过参数校验，无需再次判断

		//累计金额
		receipt.TotalAmount += one.EstimatedQuantity * one.Price

		//查询物料是否存在
		var material model.Material
		singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, bson.M{"_id": materialId})
		switch singleRes.Err() {
		case nil: //物料存在
			if e = singleRes.Decode(&material); e != nil {
				fmt.Printf("[Error]解析物料[%s]:%s\n", one.Id, e.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //物料不存在
			fmt.Printf("[Error]物料[%s]不存在\n", one.Id)
			resp.Code = http.StatusBadRequest
			resp.Msg = "物料不存在"
			return resp, nil
		default: //其他错误
			fmt.Printf("[Error]查询物料[%s]是否存在:%s\n", one.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		//收集物料列表
		im := model.InboundMaterial{
			Id:                one.Id,
			Index:             one.Index,
			Price:             one.Price,
			WarehouseId:       strings.TrimSpace(one.WarehouseId),
			WarehouseZoneId:   strings.TrimSpace(one.WarehouseZoneId),
			WarehouseRackId:   strings.TrimSpace(one.WarehouseRackId),
			WarehouseBinId:    strings.TrimSpace(one.WarehouseBinId),
			Name:              material.Name,
			Model:             material.Model,
			EstimatedQuantity: one.EstimatedQuantity,
			ActualQuantity:    one.ActualQuantity,
			Unit:              material.Unit,
		}

		//校验仓库、库区、货架、货位
		if strings.TrimSpace(one.WarehouseId) != "" { //设置了仓库
			//仓库是否查询过
			if _, ok := warehousesStatus[strings.TrimSpace(one.WarehouseId)]; !ok {
				//查询仓库状态
				singleRes = l.svcCtx.WarehouseModel.FindOne(l.ctx, bson.M{"_id": warehouseId, "status": bson.M{"$ne": code.WarehouseStatusCode("删除")}})
				switch singleRes.Err() {
				case nil: //仓库存在
					var warehouse = model.Warehouse{}
					if e = singleRes.Decode(&warehouse); e != nil {
						fmt.Printf("[Error]解析仓库[%s]:%s\n", one.WarehouseId, e.Error())
						resp.Code = http.StatusInternalServerError
						resp.Msg = "服务内部错误"
						return resp, nil
					}

					if warehouse.Status != code.WarehouseStatusCode("激活") {
						resp.Code = http.StatusBadRequest
						//仓库[暂存库]盘点中，请选择其他仓库
						resp.Msg = fmt.Sprintf("仓库[%s]%s，请选择其他仓库。", warehouse.Name, code.WarehouseStatusText(warehouse.Status))
						return resp, nil
					}

					//记录符合条件的仓库id
					warehousesStatus[warehouse.Id.Hex()] = warehouse

				case mongo.ErrNoDocuments: //仓库不存在
					fmt.Printf("[Error]仓库[%s]不存在\n", one.WarehouseId)
					resp.Code = http.StatusBadRequest
					resp.Msg = "仓库不存在"
					return resp, nil
				default: //其他错误
					fmt.Printf("[Error]查询仓库[%s]是否存在:%s\n", one.WarehouseId, err.Error())
					resp.Code = http.StatusInternalServerError
					resp.Msg = "服务内部错误"
					return resp, nil
				}
			}
			im.WarehouseName = warehousesStatus[strings.TrimSpace(one.WarehouseId)].Name
			fmt.Println("仓库名称：", warehousesStatus[strings.TrimSpace(one.WarehouseId)].Name)

		}

		if strings.TrimSpace(one.WarehouseZoneId) != "" { //设置了库区
			//库区是否查询过
			if _, ok := warehouseZonesStatus[strings.TrimSpace(one.WarehouseZoneId)]; !ok {
				//查询库区状态
				singleRes = l.svcCtx.WarehouseZoneModel.FindOne(l.ctx, bson.M{"_id": zoneId, "warehouse_id": warehouseId, "status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")}})
				switch singleRes.Err() {
				case nil: //库区存在
					var zone = model.WarehouseZone{}
					if e = singleRes.Decode(&zone); e != nil {
						fmt.Printf("[Error]解析库区[%s]:%s\n", one.WarehouseId, e.Error())
						resp.Code = http.StatusInternalServerError
						resp.Msg = "服务内部错误"
						return resp, nil
					}

					if zone.Status != code.WarehouseZoneStatusCode("激活") {
						resp.Code = http.StatusBadRequest
						//库区[暂存库]盘点中，请选择其他库区
						resp.Msg = fmt.Sprintf("库区[%s]%s，请选择其他库区。", zone.Name, code.WarehouseStatusText(zone.Status))
						return resp, nil
					}

					//记录符合条件的库区id
					warehouseZonesStatus[zone.Id.Hex()] = zone

				case mongo.ErrNoDocuments: //库区不存在
					fmt.Printf("[Error]库区[%s]不存在\n", one.WarehouseZoneId)
					resp.Code = http.StatusBadRequest
					resp.Msg = "库区不存在"
					return resp, nil
				default: //其他错误
					fmt.Printf("[Error]查询库区[%s]是否存在:%s\n", one.WarehouseZoneId, err.Error())
					resp.Code = http.StatusInternalServerError
					resp.Msg = "服务内部错误"
					return resp, nil
				}
			}

			im.WarehouseZoneName = warehouseZonesStatus[strings.TrimSpace(one.WarehouseZoneId)].Name
			fmt.Println("库区名称：", warehouseZonesStatus[strings.TrimSpace(one.WarehouseZoneId)].Name)
		}

		if strings.TrimSpace(one.WarehouseRackId) != "" { //设置了货架
			//货架是否查询过
			if _, ok := warehouseRacksStatus[strings.TrimSpace(one.WarehouseRackId)]; !ok {
				//查询货架状态
				singleRes = l.svcCtx.WarehouseRackModel.FindOne(l.ctx, bson.M{"_id": rackId, "warehouse_id": warehouseId, "warehouse_zone_id": zoneId, "status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")}})
				switch singleRes.Err() {
				case nil: //货架存在
					var rack = model.WarehouseRack{}
					if e = singleRes.Decode(&rack); e != nil {
						fmt.Printf("[Error]解析货架[%s]:%s\n", one.WarehouseRackId, e.Error())
						resp.Code = http.StatusInternalServerError
						resp.Msg = "服务内部错误"
						return resp, nil
					}

					if rack.Status != code.WarehouseRackStatusCode("激活") {
						resp.Code = http.StatusBadRequest
						//货架[暂存库]盘点中，请选择其他货架
						resp.Msg = fmt.Sprintf("货架[%s]%s，请选择其他货架。", rack.Name, code.WarehouseStatusText(rack.Status))
						return resp, nil
					}

					//记录符合条件的货架id
					warehouseRacksStatus[rack.Id.Hex()] = rack

				case mongo.ErrNoDocuments: //货架不存在
					fmt.Printf("[Error]货架[%s]不存在\n", one.WarehouseRackId)
					resp.Code = http.StatusBadRequest
					resp.Msg = "货架不存在"
					return resp, nil
				default: //其他错误
					fmt.Printf("[Error]查询货架[%s]是否存在:%s\n", one.WarehouseRackId, err.Error())
					resp.Code = http.StatusInternalServerError
					resp.Msg = "服务内部错误"
					return resp, nil
				}
			}
			im.WarehouseRackName = warehouseRacksStatus[strings.TrimSpace(one.WarehouseRackId)].Name
			fmt.Println("货架名称：", warehouseRacksStatus[strings.TrimSpace(one.WarehouseRackId)].Name)
		}

		if strings.TrimSpace(one.WarehouseBinId) != "" { //设置了货位
			//货位是否查询过
			if _, ok := warehouseBinsStatus[strings.TrimSpace(one.WarehouseBinId)]; !ok {
				//查询货位状态
				fmt.Println("查询货位状态")
				singleRes = l.svcCtx.WarehouseBinModel.FindOne(l.ctx, bson.M{"_id": binId, "warehouse_id": warehouseId, "warehouse_zone_id": zoneId, "warehouse_rack_id": rackId, "status": bson.M{"$ne": code.WarehouseBinStatusCode("删除")}})
				switch singleRes.Err() {
				case nil: //货位存在
					var bin = model.WarehouseBin{}
					if e = singleRes.Decode(&bin); e != nil {
						fmt.Printf("[Error]解析货位[%s]:%s\n", one.WarehouseBinId, e.Error())
						resp.Code = http.StatusInternalServerError
						resp.Msg = "服务内部错误"
						return resp, nil
					}

					if bin.Status != code.WarehouseBinStatusCode("激活") {
						resp.Code = http.StatusBadRequest
						//货位[暂存库]盘点中，请选择其他货位
						resp.Msg = fmt.Sprintf("货位[%s]%s，请选择其他货位。", bin.Name, code.WarehouseStatusText(bin.Status))
						return resp, nil
					}

					//记录符合条件的货位id
					warehouseBinsStatus[bin.Id.Hex()] = bin

				case mongo.ErrNoDocuments: //货位不存在
					fmt.Printf("[Error]货位[%s]不存在\n", one.WarehouseBinId)
					resp.Code = http.StatusBadRequest
					resp.Msg = "货位不存在"
					return resp, nil
				default: //其他错误
					fmt.Printf("[Error]查询货位[%s]是否存在:%s\n", one.WarehouseBinId, err.Error())
					resp.Code = http.StatusInternalServerError
					resp.Msg = "服务内部错误"
					return resp, nil
				}
			}
			im.WarehouseBinName = warehouseBinsStatus[strings.TrimSpace(one.WarehouseBinId)].Name
			fmt.Println("货位名称：", warehouseBinsStatus[strings.TrimSpace(one.WarehouseBinId)].Name)
		}

		materials = append(materials, im)

		//收集物料价格
		if one.Price > 0 {
			var update = bson.M{
				"$set": bson.M{
					"material": one.Id,
					"price":    one.Price,
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
	if req.TotalAmount > 0 {
		receipt.TotalAmount = req.TotalAmount
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
