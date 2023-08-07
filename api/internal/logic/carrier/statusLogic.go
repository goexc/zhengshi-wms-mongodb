package carrier

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatusLogic {
	return &StatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StatusLogic) Status(req *types.CarrierStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.承运商是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]承运商[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "承运商参数错误"
		return resp, nil
	}
	//排除已删除的承运商
	filter := bson.M{"_id": id, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.CarrierModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询承运商[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "承运商不存在"
		return resp, nil
	}

	//2.更改承运商状态
	var update = bson.M{
		"$set": bson.M{
			"status": req.Status,
		},
	}
	_, err = l.svcCtx.CarrierModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改承运商[%s]状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
