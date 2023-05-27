package menu

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

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

func (l *ListLogic) List() (resp *types.MenusResponse, err error) {
	resp = new(types.MenusResponse)

	//1.菜单列表查询
	filter := bson.M{}
	option := options.Find().SetSort(bson.M{"sort_id": 1})
	cur, err := l.svcCtx.MenuModel.Find(l.ctx, filter, option)
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

	//2.菜单分组
	var list = make([]types.Menu, 0)
	for _, one := range menus {
		list = append(list, types.Menu{
			Id:         one.Id.Hex(),
			Type:       one.Type,
			SortId:     one.SortId,
			ParentId:   one.ParentId,
			Path:       one.Path,
			Name:       one.Name,
			Component:  one.Component,
			Icon:       one.Icon,
			Transition: one.Transition,
			Hidden:     one.Hidden,
			Fixed:      one.Fixed,
			Perms:      one.Perms,
		})
	}
	resp.Data = list
	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
