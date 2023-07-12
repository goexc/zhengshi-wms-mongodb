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

type ProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProfileLogic {
	return &ProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProfileLogic) Profile() (resp *types.ProfileResponse, err error) {
	resp = new(types.ProfileResponse)

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
	singleRes := l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //个人存在
	case mongo.ErrNoDocuments: //个人不存在
		fmt.Printf("[Error]个人[%s]不存在\n", uid)
		resp.Code = http.StatusBadRequest
		resp.Msg = "个人不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询个人[%s]是否存在:%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var profile model.User
	if err = singleRes.Decode(&profile); err != nil {
		fmt.Printf("[Error]解析个人信息:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", uid), []string{"6472101cbf3beb2563ff8fa0"})

	//2.查询个人角色列表
	var casbinRoles []string
	casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", profile.Id.Hex()))

	fmt.Println("用户id：", profile.Id.Hex())
	fmt.Println("角色：", casbinRoles)
	var rolesId bson.A
	for _, casbinRole := range casbinRoles {
		roleId, e := primitive.ObjectIDFromHex(strings.TrimPrefix(casbinRole, "role_"))
		if e != nil {
			fmt.Printf("[Error]角色[%s]解码失败：%s\n", casbinRole, e.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		rolesId = append(rolesId, roleId)
	}

	if len(rolesId) == 0 {
		fmt.Printf("[Error]用户[%s]未分配角色\n", profile.Name)
		resp.Code = http.StatusBadRequest
		resp.Msg = "用户未分配角色"
		return resp, nil
	}

	var roleFilter = bson.M{"_id": bson.M{"$in": rolesId}}
	cur, err := l.svcCtx.RoleModel.Find(l.ctx, roleFilter)
	if err != nil {
		fmt.Printf("[Error]查询角色列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var roles []model.Role
	if err = cur.All(l.ctx, &roles); err != nil {
		fmt.Printf("[Error]解析角色列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Data.RolesId = make([]types.ProfileRole, 0)
	for _, role := range roles {
		resp.Data.RolesId = append(resp.Data.RolesId, types.ProfileRole{
			RoleId:   role.Id.Hex(),
			RoleName: role.Name,
		})

	}

	//3.路由权限列表
	// 构建聚合管道
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"role_id": bson.M{
					"$in": rolesId,
				},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "menu",
				"localField":   "menu_id",
				"foreignField": "_id",
				"as":           "menu",
			},
		},
		{
			"$unwind": "$menu",
		},
		// 筛选 parent_id 不为空的 menu
		{"$match": bson.M{"menu.parent_id": bson.M{"$ne": ""}}},
		{
			"$project": bson.M{
				//"_id":     0,
				//"role_id": 1,
				"menu": 1, //取出menu表的所有字段
				//"menu.type":  1,
				//"menu.name":  1,
				//"menu.icon":  1,
				//"menu.perms": 1,
			},
		},
	}
	cur, err = l.svcCtx.RoleMenuModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询路由列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var list = make([]types.Route, 0)
	for cur.Next(l.ctx) {
		var one bson.M
		e := cur.Decode(&one)
		if e != nil {
			fmt.Printf("[Error]解析菜单结果:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		list = append(list, types.Route{
			Id:         one["menu"].(bson.M)["_id"].(primitive.ObjectID).Hex(),
			Type:       one["menu"].(bson.M)["type"].(int64),
			SortId:     one["menu"].(bson.M)["sort_id"].(int64),
			ParentId:   one["menu"].(bson.M)["parent_id"].(string),
			Path:       one["menu"].(bson.M)["path"].(string),
			Name:       one["menu"].(bson.M)["name"].(string),
			Component:  one["menu"].(bson.M)["component"].(string),
			Icon:       one["menu"].(bson.M)["icon"].(string),
			Transition: one["menu"].(bson.M)["transition"].(string),
			Hidden:     one["menu"].(bson.M)["hidden"].(bool),
			Fixed:      one["menu"].(bson.M)["fixed"].(bool),
			IsFull:     one["menu"].(bson.M)["is_full"].(bool),
			//IsFull:    one["menu"].(bson.M)["hidden"].(bool),
			Perms:     one["menu"].(bson.M)["perms"].(string),
			Remark:    one["menu"].(bson.M)["remark"].(string),
			Children:  make([]*types.Route, 0),
			CreatedAt: one["menu"].(bson.M)["created_at"].(int64),
			UpdatedAt: one["menu"].(bson.M)["updated_at"].(int64),
		})
	}

	var routesMap = make(map[string]*types.Route, 0)

	//遍历查询结果
	for i, one := range list {
		switch true {
		case one.Type == 1: //路由权限
			//遍历 list 切片，将每个 Menu 添加到 map 中
			routesMap[list[i].Id] = &list[i]
		case one.Type == 2: //按钮权限
			resp.Data.Buttons = append(resp.Data.Buttons, types.Button{
				Name:  one.Name,
				Icon:  one.Icon,
				Perms: one.Perms,
			})
		default:
			//什么都不做
		}
	}
	// 4.遍历 list 切片，构建树形结构
	var rootRoute = make([]*types.Route, 0)
	for i := range list {
		if list[i].Type != 1 {
			continue
		}

		if parent, ok := routesMap[list[i].ParentId]; ok {
			parent.Children = append(parent.Children, &list[i])
		} else {
			rootRoute = append(rootRoute, &list[i])
		}
	}

	//5.汇总
	resp.Data.Routes = rootRoute
	resp.Data.Name = profile.Name
	resp.Data.Sex = profile.Sex
	resp.Data.DepartmentId = profile.DepartmentId
	resp.Data.DepartmentName = profile.DepartmentName
	resp.Data.Avatar = profile.Avatar
	resp.Data.Mobile = profile.Mobile
	resp.Data.Email = profile.Email
	resp.Data.Status = profile.Status
	resp.Data.Remark = profile.Remark
	resp.Data.CreatedAt = profile.CreatedAt
	resp.Data.UpdatedAt = profile.UpdatedAt

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
