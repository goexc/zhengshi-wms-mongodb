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

type RolesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RolesLogic {
	return &RolesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RolesLogic) Roles(req *types.UserRolesRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	if strings.TrimSpace(req.Id) == "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择用户"
		return resp, nil
	}

	// 1.用户是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]用户[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //用户存在
	case mongo.ErrNoDocuments: //用户不存在
		fmt.Printf("[Error]用户[%s]不存在\n", req.Id)
		resp.Code = http.StatusBadRequest
		resp.Msg = "用户不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询用户[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//2.角色是否存在
	var rolesId bson.A
	for _, one := range req.RolesId {
		roleId, e := primitive.ObjectIDFromHex(strings.TrimSpace(one))
		if e != nil {
			fmt.Printf("[Error]解析角色id[%s]：%s\n", one, e.Error())
			resp.Msg = "角色参数错误"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
		rolesId = append(rolesId, roleId)
	}

	filter = bson.M{"_id": bson.M{"$in": rolesId}}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色列表是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count != int64(len(req.RolesId)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分角色不存在"
		return resp, nil
	}

	//解绑旧角色,绑定新角色
	var roles []string
	for _, roleId := range req.RolesId {
		roles = append(roles, fmt.Sprintf("role_%s", roleId))
	}

	_, err = l.svcCtx.Enforcer.DeleteRolesForUser(fmt.Sprintf("user_%s", req.Id))
	if err != nil {
		fmt.Printf("[Error]用户[%s]清空角色:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if len(roles) > 0 {
		_, err = l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", req.Id), roles)
		if err != nil {
			fmt.Printf("[Error]用户[%s]分配角色:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
