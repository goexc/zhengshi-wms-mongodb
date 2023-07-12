package role

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MenusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MenusLogic {
	return &MenusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MenusLogic) Menus(req *types.RoleIdRequest) (resp *types.RoleMenusResponse, err error) {
	resp = new(types.RoleMenusResponse)

	//1.角色是否存在
	roleId, _ := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	var filter bson.M
	filter = bson.M{"_id": roleId}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "角色不存在"
		return resp, nil
	}

	//2.角色对应的菜单
	filter = bson.M{"role_id": roleId}
	cur, err := l.svcCtx.RoleMenuModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]菜单:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var menus []model.RoleMenu
	if err = cur.All(l.ctx, &menus); err != nil {
		fmt.Printf("[Error]解析角色[%s]菜单:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data = make([]string, 0)
	for _, one := range menus {
		resp.Data = append(resp.Data, one.MenuId.Hex())
	}
	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
