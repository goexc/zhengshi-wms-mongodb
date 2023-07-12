package user

import (
	"api/internal/svc"
	"api/internal/types"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"

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

func (l *StatusLogic) Status(req *types.UserStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	if len(req.Id) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择用户"
		return resp, nil
	}

	// 1.用户是否存在
	var ids = make([]primitive.ObjectID, 0)
	for _, id := range req.Id {
		userId, e := primitive.ObjectIDFromHex(strings.TrimSpace(id))
		if e != nil {
			fmt.Printf("[Error]角色[%s]id转换：%s\n", req.Id, e.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "参数错误"
			return resp, nil
		}
		ids = append(ids, userId)
	}

	var filter = bson.M{"_id": bson.M{"$in": ids}, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.UserModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询用户[%s]是否存在:%s\n", strings.Join(req.Id, ","), err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count < int64(len(req.Id)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分用户不存在"
		return resp, nil
	}

	//2.修改用户状态
	update := bson.M{
		"$set": bson.M{
			"status": req.Status,
		},
	}

	_, err = l.svcCtx.UserModel.UpdateMany(l.ctx, filter, &update)
	if err != nil {
		fmt.Printf("[Error]更新用户[%s]状态：%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
