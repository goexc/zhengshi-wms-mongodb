package department

import (
	"api/model"
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

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.DepartmentRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.上级部门是否存在
	var parentId primitive.ObjectID
	if strings.TrimSpace(req.ParentId) != "" {
		parentId, _ = primitive.ObjectIDFromHex(req.ParentId)
		var filter = bson.M{"_id": parentId}
		parentRes := l.svcCtx.DepartmentModel.FindOne(l.ctx, filter)
		switch parentRes.Err() {
		case nil: //上级部门存在
			var parent model.Department
			err = parentRes.Decode(&parent)
			if parent.Type <= req.Type {
				resp.Code = http.StatusBadRequest
				resp.Msg = "上级部门的类型不能低于下级部门"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //上级部门不存在
			resp.Code = http.StatusBadRequest
			resp.Msg = "上级部门不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]查询上级部门[%s]:%s\n", req.ParentId, err.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	//2.查询兄弟部门是否重复：name、full_name、code
	var ds = make([]model.Department, 0)
	var filter = bson.D{
		{"parent_id", req.ParentId},
	}

	cur, err := l.svcCtx.DepartmentModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询上级部门[%s]的下级:%s\n", req.ParentId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &ds); err != nil {
		fmt.Printf("[Error]解析同级部门:%s\n", err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	fmt.Println("兄弟部门数量：", len(ds))

	for _, d := range ds {
		if d.Name == strings.TrimSpace(req.Name) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "部门名称重复"
			return resp, nil
		}

		if d.FullName == strings.TrimSpace(req.FullName) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "部门全称重复"
			return resp, nil
		}

		if d.Code == strings.TrimSpace(req.Code) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "部门编码重复"
			return resp, nil
		}
	}

	//3.部门入库
	department := bson.M{
		"type":       req.Type,
		"sort_id":    req.SortId,
		"parent_id":  req.ParentId,
		"name":       req.Name,
		"full_name":  req.FullName,
		"code":       req.Code,
		"remark":     req.Remark,
		"created_at": time.Now().Unix(),
		"updated_at": time.Now().Unix(),
	}
	_, err = l.svcCtx.DepartmentModel.InsertOne(l.ctx, &department)
	if err != nil {
		fmt.Printf("[Error]部门[%s]入库:%s\n", req.Name, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
