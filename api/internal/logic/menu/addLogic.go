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

func (l *AddLogic) Add(req *types.Menu) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.类型为菜单时，判断路径是否重复
	filter := bson.M{
		"parent_id": req.ParentId,
		"type":      req.Type,       //类型：1.菜单，2.按钮
		"path":      req.Path,       //路径
		"perms":     req.Meta.Perms, //权限标识
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
		case nil: //上级菜单存在
			//判断上级菜单type==1
			//类型：1.菜单，2.按钮
			var one model.Menu
			if e := singleRes.Decode(&one); e != nil {
				fmt.Printf("[Error]菜单解析：%s\n", e.Error())
				resp.Msg = "服务器内部错误"
				resp.Code = http.StatusInternalServerError
				return resp, nil
			}
			if one.Type != 1 {
				resp.Msg = "只能给菜单添加子菜单"
				resp.Code = http.StatusBadRequest
				return resp, nil
			}

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
		Icon:       req.Meta.Icon,
		Transition: req.Meta.Transition,
		Hidden:     req.Meta.Hidden,
		Fixed:      req.Meta.Fixed,
		IsFull:     req.Meta.IsFull,
		Perms:      req.Meta.Perms,
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
