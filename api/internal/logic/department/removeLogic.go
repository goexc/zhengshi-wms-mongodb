package department

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

func (l *RemoveLogic) Remove(req *types.DepartmentRemoveRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	// 1.参数解析
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		fmt.Printf("[Error]解析部门id：%s\n", err.Error())
		resp.Msg = "参数错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//2.部门是否有子部门
	filter := bson.D{
		{"parent_id", req.Id},
	}
	count, err := l.svcCtx.DepartmentModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]统计子部门:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	if count > 0 {
		resp.Msg = "请先删除子部门"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	//TODO:3.部门是否分配给用户

	// 4.删除部门
	filter = bson.D{
		{"_id", id},
	}
	singleRes := l.svcCtx.DepartmentModel.FindOneAndDelete(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		resp.Msg = "部门不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]查询部门：%s\n", singleRes.Err().Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
