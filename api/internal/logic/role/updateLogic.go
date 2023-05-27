package role

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.RoleRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, _ := primitive.ObjectIDFromHex(req.Id)
	var filter = bson.M{"_id": id}
	var role = model.Role{
		Name:      req.Name,
		ParentId:  req.ParentId,
		SortId:    req.SortId,
		Status:    req.Status,
		Remark:    req.Remark,
		UpdatedAt: time.Now().Unix(),
	}

	var update = bson.M{"$set": role}
	singleRes := l.svcCtx.RoleModel.FindOneAndUpdate(l.ctx, filter, update)
	switch singleRes.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		resp.Msg = "角色不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]修改角色[%s]:%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
