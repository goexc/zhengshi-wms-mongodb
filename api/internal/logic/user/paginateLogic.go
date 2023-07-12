package user

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PaginateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPaginateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PaginateLogic {
	return &PaginateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PaginateLogic) Paginate(req *types.UsersRequest) (resp *types.UsersResponse, err error) {
	resp = new(types.UsersResponse)

	name := strings.TrimSpace(req.Name)
	mobile := strings.TrimSpace(req.Mobile)
	var filter = bson.M{}
	filter["status"] = bson.M{"$ne": "删除"}

	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter["name"] = regex
	}

	if mobile != "" {
		filter["mobile"] = mobile
	}

	opt := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(req.Size).SetSkip((req.Page - 1) * req.Size)
	cur, err := l.svcCtx.UserModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询api列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var users []model.User
	if err = cur.All(l.ctx, &users); err != nil {
		fmt.Printf("[Error]解析用户列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	count, err := l.svcCtx.UserModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询用户列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//4.查询角色
	var userRolesId = make(map[string][]string) //key:用户id， value:角色id切片
	var rolesIdStr = make(map[string]struct{})  //key:角色id， value:空值
	for _, one := range users {
		var casbinRoles []string
		casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", one.Id.Hex()))
		userRolesId[one.Id.Hex()] = make([]string, 0)
		for _, casbinRole := range casbinRoles {
			userRolesId[one.Id.Hex()] = append(userRolesId[one.Id.Hex()], strings.TrimPrefix(casbinRole, "role_"))
			rolesIdStr[strings.TrimPrefix(casbinRole, "role_")] = struct{}{}
		}
	}

	//5.查询角色信息
	var userRolesName = make(map[string][]string) //key:用户id， value:角色名称切片
	var rolesId = make([]primitive.ObjectID, 0)
	for id := range rolesIdStr {
		roleId, e := primitive.ObjectIDFromHex(strings.TrimSpace(id))
		if e != nil {
			fmt.Printf("[Error]解析角色id[%s]：%s\n", id, e.Error())
			resp.Msg = "角色参数错误"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
		rolesId = append(rolesId, roleId)
	}

	var roleFilter = bson.M{"_id": bson.M{"$in": rolesId}}
	cur, err = l.svcCtx.RoleModel.Find(l.ctx, roleFilter)
	if err != nil {
		fmt.Printf("[Error]查询角色列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var roles = make(map[string]model.Role)
	for cur.Next(l.ctx) {
		var one model.Role
		if err = cur.Decode(&one); err != nil {
			fmt.Printf("[Error]解析角色:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
		roles[one.Id.Hex()] = one
	}

	for userId, rsId := range userRolesId {
		userRolesName[userId] = make([]string, 0)
		for _, roleId := range rsId {
			userRolesName[userId] = append(userRolesName[userId], roles[roleId].Name)
		}
	}

	//7.汇总
	resp.Data.Total = count
	resp.Data.List = make([]types.User, 0)
	for _, one := range users {
		resp.Data.List = append(resp.Data.List, types.User{
			Id:             one.Id.Hex(),
			Name:           one.Name,
			Sex:            one.Sex,
			DepartmentId:   one.DepartmentId,
			DepartmentName: one.DepartmentName,
			RolesId:        userRolesId[one.Id.Hex()],
			RolesName:      userRolesName[one.Id.Hex()],
			Mobile:         one.Mobile,
			Email:          one.Email,
			Status:         one.Status,
			Remark:         one.Remark,
			CreatedAt:      one.CreatedAt,
			UpdatedAt:      one.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
