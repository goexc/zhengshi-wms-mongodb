package personal

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

type MenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MenuLogic {
	return &MenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MenuLogic) Menu() (resp *types.MenusResponse, err error) {
	resp = new(types.MenusResponse)

	uid := l.ctx.Value("uid").(string)

	// 1.个人是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(uid))
	if err != nil {
		fmt.Printf("[Error]角色[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	count, err := l.svcCtx.UserModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询个人[%s]:%s\n", uid, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	if count == 0 {
		resp.Msg = "个人不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	//2.查询绑定的角色
	var casbinRoles []string
	var rolesId bson.A
	casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", uid))
	for _, casbinRole := range casbinRoles {
		rolesId = append(rolesId, strings.TrimPrefix(casbinRole, "role_"))
	}

	if len(rolesId) == 0 {
		resp.Code = http.StatusNoContent
		resp.Msg = "用户没有绑定任何角色"
		resp.Data = make([]types.Menu, 0)
		return resp, nil
	}

	//3.查询角色绑定的菜单
	filter = bson.M{"role_id": bson.M{"$in": rolesId}}
	cur, err := l.svcCtx.RoleMenuModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询角色菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var rms []model.RoleMenu
	if err = cur.All(l.ctx, &rms); err != nil {
		fmt.Println("[Error]解析角色菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if len(rms) == 0 {
		resp.Code = http.StatusNoContent
		resp.Msg = "用户角色没有绑定任何菜单"
		resp.Data = make([]types.Menu, 0)
		return resp, nil
	}

	//4.查询菜单列表
	var menusId bson.A
	for _, m := range rms {
		mId, e := primitive.ObjectIDFromHex(m.MenuId)
		if e != nil {
			fmt.Printf("[Error]菜单[%s]解析ObjectID:%s\n", m.MenuId, e.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		menusId = append(menusId, mId)
	}

	filter = bson.M{"_id": bson.M{"$in": menusId}}
	cur, err = l.svcCtx.MenuModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Menu
	if err = cur.All(l.ctx, &ms); err != nil {
		fmt.Println("[Error]解析菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if len(ms) == 0 {
		resp.Code = http.StatusNoContent
		resp.Msg = "没有绑定任何菜单"
		resp.Data = make([]types.Menu, 0)
		return resp, nil
	}

	for _, m := range ms {
		resp.Data = append(resp.Data, types.Menu{
			Id:         m.Id.Hex(),
			Type:       m.Type,
			SortId:     m.SortId,
			ParentId:   m.ParentId,
			Path:       m.Path,
			Name:       m.Name,
			Component:  m.Component,
			Icon:       m.Icon,
			Transition: m.Transition,
			Hidden:     m.Hidden,
			Fixed:      m.Fixed,
			Perms:      m.Perms,
			Remark:     m.Remark,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
