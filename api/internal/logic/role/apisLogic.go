package role

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApisLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApisLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApisLogic {
	return &ApisLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApisLogic) Apis(req *types.RoleIdRequest) (resp *types.RoleApisResponse, err error) {
	resp = new(types.RoleApisResponse)
	role := fmt.Sprintf("role_%s", req.Id)

	perms, _ := l.svcCtx.Enforcer.GetPermissionsForUser(role)
	fmt.Println("权限列表：", perms)
	fmt.Println("权限列表数量：", len(perms))

	//uri,method数组
	var ums bson.A
	for _, perm := range perms {
		ums = append(ums, bson.M{"uri": perm[1], "method": perm[2]})
	}

	if len(ums) == 0 {
		resp.Data = make([]string, 0)
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	var filter = bson.M{"$or": ums}
	cur, err := l.svcCtx.ApiModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]api列表:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var apis []model.Api
	if err = cur.All(l.ctx, &apis); err != nil {
		fmt.Printf("[Error]解析角色[%s]api列表:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	for _, api := range apis {
		resp.Data = append(resp.Data, api.Id.Hex())
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
