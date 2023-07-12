package role

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (l *AddLogic) Add(req *types.RoleRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	if err != nil {
		fmt.Printf("[Error]用户id解析:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//1.角色名称是否重复
	var filter bson.M
	filter = bson.M{"name": strings.TrimSpace(req.Name)}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]是否占用:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "角色名称已占用"
		return resp, nil
	}

	//2.上级角色是否存在
	if strings.TrimSpace(req.ParentId) != "" {
		parentId, _ := primitive.ObjectIDFromHex(strings.TrimSpace(req.ParentId))
		filter = bson.M{"_id": parentId}
		count, err = l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
		if err != nil {
			fmt.Printf("[Error]查询上级角色[%s]是否存在:%s\n", req.ParentId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		if count == 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "上级角色不存在"
			return resp, nil
		}
	}

	//3.添加角色
	role := model.Role{
		Name:      req.Name,
		ParentId:  req.ParentId,
		Status:    req.Status,
		Remark:    req.Remark,
		CreatedBy: uid,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_, err = l.svcCtx.RoleModel.InsertOne(l.ctx, &role)
	if err != nil {
		fmt.Printf("[Error]新角色[%s]入库:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
