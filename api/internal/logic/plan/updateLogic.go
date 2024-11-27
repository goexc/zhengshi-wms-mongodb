package plan

import (
	"api/internal/svc"
	"api/internal/types"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

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

func (l *UpdateLogic) Update(req *types.PlanUpdateRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.查询并修改计划
	planId, _ := primitive.ObjectIDFromHex(req.Id)
	var update = bson.M{"$set": bson.M{
		"material_quantity": req.MaterialQuantity,
		"deadline":          req.Deadline,
	}}

	singleRes := l.svcCtx.PlanModel.FindOneAndUpdate(l.ctx, bson.M{"_id": planId}, &update)
	switch singleRes.Err() {
	case nil: //计划存在
	case mongo.ErrNoDocuments: //计划不存在
		fmt.Printf("[Error]计划[%s]不存在\n", req.Id)
		resp.Code = http.StatusBadRequest
		resp.Msg = "计划不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询并修改计划[%s]:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
