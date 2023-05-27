package role

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveLogic) Remove(req *types.RoleRemoveRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.角色是否存在
	//2.删除角色
	id, _ := primitive.ObjectIDFromHex(req.Id)
	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.RoleModel.FindOneAndDelete(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		resp.Msg = "角色不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]删除角色[%s]:%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
