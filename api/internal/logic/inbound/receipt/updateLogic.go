package receipt

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 可修改的入库单状态
var canUpdateStatus = map[string]string{"待审核": "", "审核不通过": ""}

func (l *UpdateLogic) Update(req *types.InboundReceiptUpdateRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//只能修改待审核、审核不通过的入库单

	var receipt model.InboundReceipt

	id, _ := primitive.ObjectIDFromHex(req.Id)
	if id.IsZero() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数id错误"
		return resp, nil
	}

	//1.入库单是否存在
	var filter = bson.M{"_id": id}
	//count, err := l.svcCtx.InboundReceiptModel.CountDocuments(l.ctx, filter)
	singleRes := l.svcCtx.InboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]入库单[%s]解析:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrClientDisconnected:
		fmt.Println("[Error]查询入库单：MongoDB连接断开")
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	case mongo.ErrNoDocuments:
		fmt.Printf("[Error]入库单[%s]不存在\n", singleRes.Err().Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]入库单查询：%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if _, ok := canUpdateStatus[code.InboundReceiptStatusText(receipt.Status)]; !ok {
		resp.Code = http.StatusBadRequest
		resp.Msg = "只能修改'待审核'、'审核不通过'的入库单"
		return resp, nil
	}

	//2.供应商是否存在
	var supplierId, supplierName string
	if req.Type != "退货入库" {
		var supplier model.Supplier
		sId, _ := primitive.ObjectIDFromHex(req.SupplierId)
		singleRes = l.svcCtx.SupplierModel.FindOne(l.ctx, bson.M{"_id": sId})
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

		supplierId = supplier.Id.Hex()
		supplierName = supplier.Name
	}

	//3.客户是否存在
	var customerId, customerName string
	if req.Type == "退货入库" {
		var customer model.Customer
		cId, _ := primitive.ObjectIDFromHex(req.CustomerId)
		singleRes = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": cId})
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

		customerId = customer.Id.Hex()
		customerName = customer.Name
	}

	//3.仓库、库区、货架、货位是否存在
	var materials = make([]model.InboundMaterial, 0)                //入库单的物料列表
	var warehousesStatus = make(map[string]model.Warehouse)         //仓库id=>“”
	var warehouseZonesStatus = make(map[string]model.WarehouseZone) //库区id=>“”
	var warehouseRacksStatus = make(map[string]model.WarehouseRack) //货架id=>“”
	var warehouseBinsStatus = make(map[string]model.WarehouseBin)   //货位id=>“”
	var totalAmount float64
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
		totalAmount += one.EstimatedQuantity * one.Price

		//查询物料是否存在
		var material model.Material
		singleRes = l.svcCtx.MaterialModel.FindOne(l.ctx, bson.M{"_id": materialId})
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

		fmt.Printf("[Info]物料[%s]仓储位置：%s/%s/%s/%s\n", one.Id, one.WarehouseId, one.WarehouseZoneId, one.WarehouseRackId, one.WarehouseBinId)
		//收集物料列表
		im := model.InboundMaterial{
			Id:                one.Id,
			Index:             one.Index,
			Status:            one.Status,
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

	}

	//如果请求参数中的总金额>0，那么使用请求参数中的总金额
	if req.TotalAmount > 0 {
		totalAmount = req.TotalAmount
	}

	//4.不更新入库单编号[code]、入库单状态[status]、入库单类型[type]
	var update = bson.M{
		"$set": bson.M{
			"status":         code.InboundReceiptStatusCode("待审核"), //待审核、审核不通过的入库单，统一修改为待审核状态
			"type":           req.Type,
			"supplier_id":    supplierId,
			"supplier_name":  supplierName,
			"customer_id":    customerId,
			"customer_name":  customerName,
			"total_amount":   totalAmount,
			"receiving_date": req.ReceivingDate,
			"materials":      materials,
			"annex":          req.Annex,
			"remark":         strings.TrimSpace(req.Remark),
			"editor_id":      l.ctx.Value("uid").(string),
			"editor_name":    l.ctx.Value("name").(string),
			"updated_at":     time.Now().Unix(),
		},
	}
	_, err = l.svcCtx.InboundReceiptModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改入库单[%s]信息：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
