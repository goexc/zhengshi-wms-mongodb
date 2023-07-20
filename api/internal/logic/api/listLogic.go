package api

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"

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

func (l *ListLogic) List() (resp *types.ApisResponse, err error) {
	resp = new(types.ApisResponse)
	resp.Data = make([]*types.Api, 0)

	var apis []model.Api
	cur, err := l.svcCtx.ApiModel.Find(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询api列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &apis); err != nil {
		fmt.Printf("[Error]解析api列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var list []types.Api
	for _, one := range apis {
		list = append(list, types.Api{
			Id:        one.Id.Hex(),
			Type:      one.Type,
			SortId:    one.SortId,
			ParentId:  one.ParentId,
			Uri:       one.Uri,
			Method:    one.Method,
			Name:      one.Name,
			Remark:    one.Remark,
			CreatedAt: one.CreatedAt,
			UpdatedAt: one.UpdatedAt,
		})
	}

	var apiMap = make(map[string]*types.Api)
	for i, one := range list {
		apiMap[one.Id] = &list[i]
	}

	var rootApi = make([]*types.Api, 0)
	for i := range list {
		if parent, ok := apiMap[list[i].ParentId]; ok {
			parent.Children = append(parent.Children, &list[i])
		} else {
			rootApi = append(rootApi, &list[i])
		}
	}

	resp.Data = rootApi
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
