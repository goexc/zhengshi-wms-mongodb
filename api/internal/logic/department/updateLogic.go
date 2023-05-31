package department

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"
	"time"
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

func (l *UpdateLogic) Update(req *types.DepartmentRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.部门、上级部门是否存在
	if strings.TrimSpace(req.Id) == "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部门id不能为空"
		return resp, nil
	}

	var ids bson.A
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]部门[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	if strings.TrimSpace(req.ParentId) != "" {
		parentId, _ := primitive.ObjectIDFromHex(req.ParentId)
		ids = bson.A{id, parentId}
	} else {
		ids = bson.A{id}
	}

	filter := bson.D{
		{"_id", bson.D{{"$in", ids}}},
	}

	cur, err := l.svcCtx.DepartmentModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]部门id查询失败：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var count int
	for cur.Next(l.ctx) {
		count++
	}

	if err = cur.Err(); err != nil {
		fmt.Printf("[Error]部门id读取失败：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if count != len(ids) {
		fmt.Printf("[Error]部门不存在：count[%d],ids[%d]\n", count, len(ids))
		resp.Code = http.StatusBadRequest
		resp.Msg = "部门不存在"
		return resp, nil
	}

	//2.更新部门信息
	filter = bson.D{{"_id", id}}
	department := model.Department{
		Type:      req.Type,
		SortId:    req.SortId,
		ParentId:  req.ParentId,
		Name:      req.Name,
		Code:      req.Code,
		Remark:    req.Remark,
		UpdatedAt: time.Now().Unix(),
	}

	update := bson.D{
		{"$set", department},
	}
	_, err = l.svcCtx.DepartmentModel.UpdateOne(l.ctx, filter, update)
	if err != nil {
		fmt.Printf("[Error]更新部门：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
