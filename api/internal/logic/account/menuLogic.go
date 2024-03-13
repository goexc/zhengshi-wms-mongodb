package account

import (
	"api/model"
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

func (l *MenuLogic) Menu() (resp *types.AccountPermsResponse, err error) {
	resp = new(types.AccountPermsResponse)

	uid := l.ctx.Value("uid").(string)

	// 1.账号是否存在
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
		fmt.Printf("[Error]查询账号[%s]:%s\n", uid, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	if count == 0 {
		resp.Msg = "账号不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	//2.查询绑定的角色
	var isSystemRole bool //超级管理员标记
	var casbinRoles []string
	var rolesId bson.A
	casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", uid))
	for _, casbinRole := range casbinRoles {
		if strings.TrimPrefix(casbinRole, "role_") == l.svcCtx.Config.Ids.Role {
			isSystemRole = true
		}
		one, e := primitive.ObjectIDFromHex(strings.TrimPrefix(casbinRole, "role_"))
		if e != nil {
			fmt.Printf("用户角色[%s]格式错误：%s\n", casbinRole, e.Error())
			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}

		rolesId = append(rolesId, one)
	}

	if len(rolesId) == 0 {
		resp.Code = http.StatusNoContent
		resp.Msg = "用户没有绑定任何角色"
		return resp, nil
	}

	fmt.Println("用户角色：", rolesId)
	fmt.Println("系统管理员：", isSystemRole)

	//3.查询角色绑定的菜单
	var cur *mongo.Cursor
	if !isSystemRole {
		filter = bson.M{"role_id": bson.M{"$in": rolesId}}
		cur, err = l.svcCtx.RoleMenuModel.Find(l.ctx, filter)
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
			return resp, nil
		}

		//4.查询菜单列表
		var menusId = make([]primitive.ObjectID, 0)
		for _, rm := range rms {
			menusId = append(menusId, rm.MenuId)
		}
		filter = bson.M{"_id": bson.M{"$in": menusId}}
	} else {
		filter = bson.M{}
	}

	//option := options.Find().SetSort(bson.M{"sort_id": 1})
	//cur, err = l.svcCtx.MenuModel.Find(l.ctx, filter, option)
	cur, err = l.svcCtx.MenuModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var menus []model.Menu
	if err = cur.All(l.ctx, &menus); err != nil {
		fmt.Println("[Error]解析菜单列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	fmt.Println("菜单数量：", len(menus))

	//5.菜单分组
	var list = make([]types.Menu, 0)
	for _, one := range menus {
		switch one.Type {
		case 1: //菜单
			list = append(list, types.Menu{
				Id:        one.Id.Hex(),
				Type:      one.Type,
				SortId:    one.SortId,
				ParentId:  one.ParentId,
				Path:      one.Path,
				Name:      one.Name,
				Component: one.Component,
				Meta: types.MetaProps{
					Title:      one.Title,
					Icon:       one.Icon,
					Transition: one.Transition,
					Hidden:     one.Hidden,
					Fixed:      one.Fixed,
					IsFull:     one.IsFull,
					Perms:      one.Perms,
				},
				CreatedAt: one.CreatedAt,
				UpdatedAt: one.UpdatedAt,
			})
		case 2: //按钮
			resp.Data.Buttons = append(resp.Data.Buttons, types.Button{
				Name:  one.Name,
				Icon:  one.Icon,
				Perms: one.Perms,
			})
		default:
			fmt.Printf("未知菜单类型：%#v\n", one)
		}
	}

	//6.构造树形数据结构
	menuMap := make(map[string]*types.Menu)

	//遍历 menus 切片，将每个 Menu 添加到 map 中
	for i := range list {
		menuMap[list[i].Id] = &list[i]
	}

	// 遍历 list 切片，构建树形结构
	var rootMenu = make([]*types.Menu, 0)
	for i := range list {
		if parent, ok := menuMap[list[i].ParentId]; ok {
			//fmt.Println("子菜单：", list[i].Name, ", parentId:", list[i].ParentId)
			parent.Children = append(parent.Children, &list[i])
		} else {
			//fmt.Println("顶级菜单：", list[i].Name, ", parentId:", list[i].ParentId, ", id:", list[i].Id)
			rootMenu = append(rootMenu, &list[i])
		}
	}

	resp.Data.Menus = rootMenu
	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
