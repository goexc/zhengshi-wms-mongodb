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

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *types.UserListRequest) (resp *types.UserListResponse, err error) {
	resp = new(types.UserListResponse)

	account := strings.TrimSpace(req.Account)
	mobile := strings.TrimSpace(req.Mobile)
	var filter bson.M
	switch true {
	case account != "" && mobile != "":
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + account + ".*", Options: "i"}}
		filter = bson.M{"account": regex, "mobile": mobile}
	case account != "":
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + account + ".*", Options: "i"}}
		filter = bson.M{"account": regex}
	case mobile != "":
		filter = bson.M{"mobile": mobile}
	default:
	}
	fmt.Println("过滤器：", filter)

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
	//TODO:4.查询角色
	var userRoles = make(map[string][]string) //key:用户id， value:角色id切片
	for _, one := range users {
		var casbinRoles []string
		casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", one.Id.Hex()))
		fmt.Println("角色：", casbinRoles)
		for _, casbinRole := range casbinRoles {
			userRoles[one.Id.Hex()] = append(userRoles[one.Id.Hex()], strings.TrimPrefix(casbinRole, "role_"))
		}
	}

	fmt.Println("角色列表：", userRoles)
	//5.汇总
	resp.Data.Total = count
	resp.Data.List = make([]types.User, 0)
	for _, one := range users {
		resp.Data.List = append(resp.Data.List, types.User{
			Id:             one.Id.Hex(),
			Account:        one.Account,
			Password:       one.Password,
			Sex:            one.Sex,
			DepartmentId:   one.DepartmentId,
			DepartmentName: one.DepartmentName,
			RolesId:        userRoles[one.Id.Hex()],
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
