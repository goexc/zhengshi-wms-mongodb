package customer

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

func (l *StatusLogic) Status(req *types.CustomerStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.客户是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]客户[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户参数错误"
		return resp, nil
	}
	//排除已删除的客户
	filter := bson.M{"_id": id, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.CustomerModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	//2.更改客户状态
	var update = bson.M{
		"$set": bson.M{
			"status": req.Status,
		},
	}
	_, err = l.svcCtx.CustomerModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改客户[%s]状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
