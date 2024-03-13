package outbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PackLogic {
	return &PackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PackLogic) Pack(req *types.OutboundOrderPackRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.打包时间不能超过当前时间
	if req.PackingTime > time.Now().Unix() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "打包时间不能超过当前时间"
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
	if order.Status != "已拣货" {
		switch order.Status {
		case "预发货", "待拣货":
			resp.Msg = fmt.Sprintf("%s出库单无法打包", order.Status)
			resp.Code = http.StatusBadRequest
			return resp, nil
		case "已拣货":
		case "已称重":
			if order.IsPack == 1 {
				resp.Msg = "不能重复打包"
				resp.Code = http.StatusBadRequest
				return resp, nil
			}
		default:
			resp.Msg = "不能重复打包"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
	}

	//2.修改发货单状态：已拣货->已打包
	var update = bson.M{
		"$set": bson.M{
			"status":       "已打包",
			"is_pack":      1,
			"packing_time": req.PackingTime,
		},
	}
	_, err = l.svcCtx.OutboundOrderModel.UpdateOne(l.ctx, bson.M{"code": strings.TrimSpace(req.Code)}, update)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]状态(已打包)：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
