package menu

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.MenuRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.路径是否重复
	filter := bson.M{
		"path": req.Path,
	}
	singleRes := l.svcCtx.MenuModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Menu
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复菜单:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if one.Path == strings.TrimSpace(req.Path) {
			resp.Msg = "菜单路径重复"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil

	case mongo.ErrNoDocuments: //菜单未占用
	default:
		fmt.Printf("[Error]查询重复菜单:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.上级菜单是否存在
	if strings.TrimSpace(req.ParentId) != "" {
		parentId, err := primitive.ObjectIDFromHex(req.ParentId)
		if err != nil {
			fmt.Printf("[Error]解析上级菜单id：%s\n", err.Error())
			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}
		filter = bson.M{"_id": parentId}
		singleRes = l.svcCtx.MenuModel.FindOne(l.ctx, filter)
		switch singleRes.Err() {
		case nil:
		case mongo.ErrNoDocuments: //上级菜单不存在
			fmt.Printf("[Error]上级菜单[%s]不存在\n", req.ParentId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "上级菜单不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]查询上级菜单:%s\n", singleRes.Err().Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	//3.添加菜单
	menu := model.Menu{
		Type:       req.Type,
		Name:       req.Name,
		Path:       req.Path,
		ParentId:   req.ParentId,
		SortId:     req.SortId,
		Component:  req.Component,
		Icon:       req.Icon,
		Transition: req.Transition,
		Hidden:     req.Hidden,
		Fixed:      req.Fixed,
		Perms:      req.Perms,
		Remark:     req.Remark,
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}
	_, err = l.svcCtx.MenuModel.InsertOne(l.ctx, &menu)
	if err != nil {
		fmt.Printf("[Error]新增菜单入库：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
