package outbound

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type WeighLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWeighLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WeighLogic {
	return &WeighLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WeighLogic) Weigh(req *types.OutboundOrderWeighRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.称重时间不能超过当前时间
	if req.WeighingTime > time.Now().Unix() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "称重时间不能超过当前时间"
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

	//1.2 出库单状态是“已拣货”
	switch order.Status {
	case "预发货", "待拣货":
		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("%s出库单无法称重", order.Status)
		return resp, nil
	case "已拣货":
	case "已打包":
		if order.IsWeigh == 1 {
			resp.Msg = "不能重复称重"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
	default:
		resp.Code = http.StatusBadRequest
		resp.Msg = "不能重复称重"
		return resp, nil
	}

	//2.修改发货单状态：已拣货、已打包->已称重
	var set = bson.M{
		"$set": bson.M{
			"status":        "已称重",
			"is_weigh":      1,
			"weighing_time": req.WeighingTime,
		},
	}
	_, err = l.svcCtx.OutboundOrderModel.UpdateOne(l.ctx, bson.M{"code": strings.TrimSpace(req.Code)}, set)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]状态(已称重)：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.批量更新物料重量
	// 3.3.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)
	for _, one := range req.Materials {
		filter := bson.D{{"material_id", one.MaterialId}, {"order_code", req.Code}}
		update := bson.D{
			{
				"$set", bson.D{{"weight", one.Weight}},
			},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//3.3.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	_, err = l.svcCtx.OutboundMaterialModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]出库单[%s]称重批量更新：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
