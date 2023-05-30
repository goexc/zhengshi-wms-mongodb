package role

import (
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

func (l *StatusLogic) Status(req *types.RoleStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]角色[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	var update = bson.M{"$set": bson.M{
		"status":     req.Status,
		"updated_at": time.Now().Unix(),
	}}
	singleRes := l.svcCtx.RoleModel.FindOneAndUpdate(l.ctx, filter, update)
	switch singleRes.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		resp.Msg = "角色不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]修改角色[%s]状态:%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
