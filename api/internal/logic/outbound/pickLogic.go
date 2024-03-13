package outbound

import (
	"api/model"
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

type PickLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPickLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PickLogic {
	return &PickLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PickLogic) Pick(req *types.OutboundOrderPickRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.拣货时间不能超过当前时间
	if req.PickingTime > time.Now().Unix() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "拣货时间不能超过当前时间"
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

	//1.2 出库单状态是“待拣货”
	if order.Status != "待拣货" {
		switch order.Status {
		case "预发货":
			resp.Msg = "出库单未确认，无法拣货"
		case "待拣货":
		default:
			resp.Msg = "不能重复拣货"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	//2.查询发货物料及其出库数量
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": strings.TrimSpace(req.Code)})
	if err != nil {
		fmt.Printf("[Error]查询发货单[%s]物料列表：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials []model.OutboundOrderMaterial
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析发货单[%s]物料列表：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//3.批量更新库存锁定数量、可用数量
	// 3.3.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)

	for _, material := range materials {
		for _, inventory := range material.Inventorys {
			inventoryId, _ := primitive.ObjectIDFromHex(inventory.InventoryId)
			filter := bson.D{{"_id", inventoryId}}
			update := bson.D{
				{"$inc", bson.D{
					{"locked_quantity", -inventory.ShipmentQuantity},
					{"available_quantity", -inventory.ShipmentQuantity},
				}},
			}

			bulkWrite := mongo.NewUpdateOneModel()
			bulkWrite.SetFilter(filter)
			bulkWrite.SetUpdate(update)

			bulkWrites = append(bulkWrites, bulkWrite)
		}
	}

	//3.3.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	_, err = l.svcCtx.InventoryModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]出库单[%s]拣货批量更新库存：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.修改发货单状态：待拣货->已拣货
	var update = bson.M{
		"$set": bson.M{
			"status":       "已拣货",
			"picking_time": req.PickingTime,
		},
	}
	_, err = l.svcCtx.OutboundOrderModel.UpdateOne(l.ctx, bson.M{"code": strings.TrimSpace(req.Code)}, update)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]状态(已拣货)：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
