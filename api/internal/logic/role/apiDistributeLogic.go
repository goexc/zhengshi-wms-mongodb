package role

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
)

type ApiDistributeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApiDistributeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApiDistributeLogic {
	return &ApiDistributeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApiDistributeLogic) ApiDistribute(req *types.RoleApisRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//Api权限写入casbin
	//1.角色是否存在
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		fmt.Printf("[Error]角色[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]:%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	if count == 0 {
		resp.Msg = "角色不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	//2.查询分配的api是否存在
	var apisId bson.A
	for _, apiId := range req.ApisId {
		aId, _ := primitive.ObjectIDFromHex(apiId)
		apisId = append(apisId, aId)
	}
	filter = bson.M{"_id": bson.M{"$in": apisId}}
	cur, err := l.svcCtx.ApiModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询待绑定Api是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var apis []model.Api
	if err = cur.All(l.ctx, &apis); err != nil {
		fmt.Printf("[Error]解析待绑定Api是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if len(apis) != len(req.ApisId) {
		fmt.Printf("[Error]待绑定Api数：%d，可查询Api数：%d\n", len(req.ApisId), len(apis))
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分Api不存在"
		return resp, nil
	}

	//3.删除角色绑定的api
	_, err = l.svcCtx.Enforcer.DeletePermissionsForUser(fmt.Sprintf("role_%s", strings.TrimSpace(req.Id)))
	if err != nil {
		fmt.Printf("[Error]删除角色[%s]的api:%s\n", strings.TrimSpace(req.Id), err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//3.角色添加权限
	var permissions [][]string
	for _, api := range apis {
		//3.1 角色添加api[role_menu]
		if api.Type == 1 || api.Uri == "" {
			continue
		}

		permissions = append(permissions, []string{fmt.Sprintf("role_%s", req.Id), api.Uri, api.Method})
	}

	fmt.Println("待添加权限：", permissions)
	if len(permissions) == 0 {
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}
	_, err = l.svcCtx.Enforcer.AddPolicies(permissions)
	if err != nil {
		fmt.Printf("[Error]角色[%s]添加权限:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if err = l.svcCtx.Enforcer.SavePolicy(); err != nil {
		fmt.Printf("[Error]角色[%s]保存权限:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
