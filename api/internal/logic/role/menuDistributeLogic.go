package role

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MenuDistributeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMenuDistributeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MenuDistributeLogic {
	return &MenuDistributeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MenuDistributeLogic) MenuDistribute(req *types.RoleMenusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.角色是否存在
	role_id, _ := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	var filter bson.M
	filter = bson.M{"_id": role_id}
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

	//2.菜单是否存在
	var menusId bson.A
	for _, menuId := range req.MenusId {
		mId, _ := primitive.ObjectIDFromHex(menuId)
		menusId = append(menusId, mId)
	}
	fmt.Println("待查询菜单：", menusId)
	filter = bson.M{"_id": bson.M{"$in": menusId}}
	count, err = l.svcCtx.MenuModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询待绑定菜单id是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count != int64(len(req.MenusId)) {
		fmt.Printf("[Error]待绑定菜单数：%d，可查询菜单数：%d\n", len(req.MenusId), count)
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分菜单不存在"
		return resp, nil
	}

	//3.删除角色对应的菜单
	filter = bson.M{"role_id": role_id}
	_, err = l.svcCtx.RoleMenuModel.DeleteMany(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]删除角色[%s]对应的菜单:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.角色绑定菜单
	var now = time.Now()
	var docs []interface{}
	for _, menuId := range menusId {
		docs = append(docs, model.RoleMenu{
			RoleId:    role_id,
			MenuId:    menuId.(primitive.ObjectID),
			CreatedAt: now.Unix(),
		})
	}

	_, err = l.svcCtx.RoleMenuModel.InsertMany(l.ctx, docs)
	if err != nil {
		fmt.Printf("[Error]角色[%s]绑定菜单列表:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
