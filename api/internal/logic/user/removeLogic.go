package user

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

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

func (l *RemoveLogic) Remove(req *types.UserIdRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.确定用户存在并删除
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]解析用户id[%s]：%s\n", req.Id, err.Error())
		resp.Msg = "参数错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.UserModel.FindOneAndDelete(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //发现并成功删除
	case mongo.ErrNoDocuments:
		resp.Msg = "用户不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]查询并删除用户[%s]：%s\n", req.Id, singleRes.Err().Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//2.解绑旧角色
	_, err = l.svcCtx.Enforcer.DeleteRolesForUser(fmt.Sprintf("user_%s", req.Id))
	if err != nil {
		fmt.Printf("[Error]用户[%s]清空角色:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
