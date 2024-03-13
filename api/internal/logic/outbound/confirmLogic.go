package outbound

import (
	"api/model"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
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

type ConfirmLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConfirmLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmLogic {
	return &ConfirmLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfirmLogic) Confirm(req *types.OutboundOrderConfirmRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.确认时间不能超过当前时间
	if req.ConfirmTime > time.Now().Unix() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "确认时间不能超过当前时间"
		return resp, nil
	}

	//1.查询出库单
	//1.1 查询出库单
	singleRes := l.svcCtx.OutboundOrderModel.FindOne(l.ctx, bson.M{"code": req.Code})
	switch singleRes.Err() {
	case nil: //出库单存在
	case mongo.ErrNoDocuments: //出库单不存在
		fmt.Printf("[Error]出库单[%s]不存在\n", req.Code)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询出库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var order model.OutboundOrder
	if err = singleRes.Decode(&order); err != nil {
		fmt.Printf("[Error]解析出库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.2 出库单状态是“预发货”
	if order.Status != "预发货" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "无需重复确认入库单"
		return resp, nil
	}

	//2.查询出库单物料
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": req.Code})
	if err != nil {
		fmt.Printf("[Error]查询出库单物料列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var originMaterials []model.OutboundOrderMaterial
	if err = cur.All(l.ctx, &originMaterials); err != nil {
		fmt.Printf("[Error]解析出库单[%s]物料列表：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.发货数量有变动时，修改物料发货数量
	//3.1 比较发货数量，记录发货数量差值
	var materials = make([]interface{}, 0)                                //新物料列表
	var confirmMaterials = make(map[string]types.OutboundConfirmMaterial) //确认的物料列表
	var confirmQuantity = make(map[string]float64)                        //确认的物料的出货数量
	var restMaterials = make([]interface{}, 0)                            //其余的物料列表
	var inventorys = make(map[primitive.ObjectID]float64)                 //库存锁定数量
	//新增入库单的单号
	var restCode = fmt.Sprintf("O-%s-%d", time.Now().Format("20060102-15-04-05"), time.Now().UnixMilli()%1000) //YYYY-MM-DD-HH-mm-ss-SSS
	order.TotalAmount = 0                                                                                      //重置出库单总金额

	for _, one := range req.Materials {
		confirmMaterials[one.MaterialId] = one

		for _, item := range one.Inventorys {
			confirmQuantity[one.MaterialId] += item.ShipmentQuantity

			inventoryId, _ := primitive.ObjectIDFromHex(item.InventoryId)
			inventorys[inventoryId] = item.ShipmentQuantity
		}
	}

	//累计总金额
	var totalAmount float64 = 0

	for _, one := range originMaterials {
		if _, ok := confirmMaterials[one.MaterialId]; ok {
			if one.Quantity > confirmQuantity[one.MaterialId] { //出货数量小于原定出货数量：差额存入其他物料数组
				restMaterial := model.OutboundOrderMaterial{
					OrderCode:  restCode,
					MaterialId: one.MaterialId,
					Index:      one.Index,
					Price:      one.Price,
					Name:       one.Name,
					Model:      one.Model,
					Quantity:   one.Quantity - confirmQuantity[one.MaterialId],
					Unit:       one.Unit,
				}
				restMaterials = append(restMaterials, restMaterial)
			}

			one.Quantity = confirmQuantity[one.MaterialId]
			totalAmount += decimal.NewFromFloat(one.Quantity).Mul(decimal.NewFromFloat(one.Price)).InexactFloat64()

			//记录物料库存拣货数量
			for _, inventory := range confirmMaterials[one.MaterialId].Inventorys {
				one.Inventorys = append(one.Inventorys, model.OutboundMaterialInventory{
					InventoryId:      inventory.InventoryId,
					ShipmentQuantity: inventory.ShipmentQuantity,
				})
			}

			materials = append(materials, one)
			order.TotalAmount += one.Quantity * one.Price
		} else { //没有选择的物料：全部存入其他物料数组
			restMaterial := model.OutboundOrderMaterial{
				OrderCode:  restCode,
				MaterialId: one.MaterialId,
				Index:      one.Index,
				Price:      one.Price,
				Name:       one.Name,
				Model:      one.Model,
				Quantity:   one.Quantity,
				Unit:       one.Unit,
			}
			restMaterials = append(restMaterials, restMaterial)
		}
	}

	if len(materials) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择物料"
		return resp, nil
	}

	//3.2 删除旧物料列表，写入新物料列表
	deleteRes, err := l.svcCtx.OutboundMaterialModel.DeleteMany(l.ctx, bson.M{"order_code": req.Code})
	if err != nil {
		fmt.Printf("[Error]删除入库单[%s]旧物料:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if deleteRes.DeletedCount == 0 {
		fmt.Printf("[Error]没有有效删除入库单[%s]旧物料\n", req.Code)
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	_, err = l.svcCtx.OutboundMaterialModel.InsertMany(l.ctx, materials)
	if err != nil {
		fmt.Printf("[Error]确认入库单[%s]物料：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	//3.3 在对应的库存记录中，锁定相应的数量
	// 3.3.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)
	for inventoryId, shipmentQuantity := range inventorys {
		filter := bson.D{{"_id", inventoryId}}
		update := bson.D{
			{"$inc", bson.D{{"locked_quantity", shipmentQuantity}}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//3.3.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	bulkRes, err := l.svcCtx.InventoryModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]入库单[%s]批量更新库存：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	fmt.Printf("入库单[%s]更新了%d条库存\n", req.Code, bulkRes.ModifiedCount)

	//3.4 出库单状态：预发货->待拣货
	var update = bson.M{
		"$set": bson.M{
			"status":       "待拣货",
			"confirm_time": req.ConfirmTime,
			"total_amount": totalAmount,
		},
	}
	_, err = l.svcCtx.OutboundOrderModel.UpdateOne(l.ctx, bson.M{"code": strings.TrimSpace(req.Code)}, update)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]状态(待拣货)：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.4 存在发货数量差值列表时，用新的出库单保存，状态设置为【预发货】
	if len(restMaterials) > 0 {
		var restOrder = model.OutboundOrder{
			Status:        "预发货",
			Type:          order.Type,
			Code:          restCode,
			SupplierId:    order.SupplierId,
			SupplierName:  order.SupplierName,
			CustomerId:    order.CustomerId,
			CustomerName:  order.CustomerName,
			TotalAmount:   0,
			Remark:        order.Remark,
			Annex:         order.Annex,
			CreatorId:     l.ctx.Value("uid").(string),
			CreatorName:   l.ctx.Value("name").(string),
			ConfirmTime:   0,
			PickingTime:   0,
			PackingTime:   0,
			WeighingTime:  0,
			DepartureTime: 0,
			ReceiptTime:   0,
			CreatedAt:     time.Now().Unix(),
		}

		//计算出库单总金额
		for _, one := range restMaterials {
			restOrder.TotalAmount += one.(model.OutboundOrderMaterial).Quantity * one.(model.OutboundOrderMaterial).Price
		}

		_, err = l.svcCtx.OutboundOrderModel.InsertOne(l.ctx, &restOrder)
		if err != nil {
			fmt.Printf("[Error]为剩余物料创建新出库单[%s]:%s\n", restCode, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		_, err = l.svcCtx.OutboundMaterialModel.InsertMany(l.ctx, restMaterials)
		if err != nil {
			fmt.Printf("[Error]剩余物料保存到新出库单[%s]:%s\n", restCode, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
