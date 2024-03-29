package role

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

func (l *PaginateLogic) Paginate(req *types.RolesRequest) (resp *types.RolesResponse, err error) {
	resp = new(types.RolesResponse)

	name := strings.TrimSpace(req.Name)
	//1.角色分页
	var filter = bson.M{}
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter = bson.M{"name": regex}
	}
	//分页中可以显示被删除的角色
	//filter["status"] = bson.M{"$ne": "删除"}

	var opt = options.Find().SetSort(bson.M{"created_at": 1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)
	cur, err := l.svcCtx.RoleModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询角色列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var roles []model.Role
	if err = cur.All(l.ctx, &roles); err != nil {
		fmt.Println("[Error]解析角色列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.角色总数量
	total, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]角色总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Total = total
	for _, role := range roles {
		resp.Data.List = append(resp.Data.List, types.Role{
			Id:        role.Id.Hex(),
			ParentId:  role.ParentId,
			Status:    role.Status,
			Name:      role.Name,
			Remark:    role.Remark,
			CreatedAt: role.CreatedAt,
			UpdatedAt: role.UpdatedAt,
		})

	}

	return resp, nil
}
