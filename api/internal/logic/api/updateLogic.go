package api

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

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.ApiRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.api是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]api[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.ApiModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Api
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]Api解析：%s\n", err.Error())
			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}
		if one.Type != req.Type {
			resp.Msg = "不支持修改api类型"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
	case mongo.ErrNoDocuments: //api不存在
	default:
		fmt.Printf("[Error]查询Api：%s\n", singleRes.Err().Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//2.上级api是否存在
	if strings.TrimSpace(req.ParentId) != "" {
		parentId, _ := primitive.ObjectIDFromHex(strings.TrimSpace(req.ParentId))
		filter = bson.M{"_id": parentId}
		count, err := l.svcCtx.ApiModel.CountDocuments(l.ctx, filter)
		if err != nil {
			fmt.Printf("[Error]查询api上级id[%s]:%s\n", req.ParentId, err.Error())
			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}

		if count == 0 {
			resp.Msg = "上级api不存在"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
	}

	//3.更新的api是否重复
	var name = strings.TrimSpace(req.Name)
	var uri = strings.TrimSpace(req.Uri)
	var method = strings.TrimSpace(req.Method)
	if len(name) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请填写 api 名称"
		return resp, nil
	}

	filter = bson.M{
		"_id": bson.M{
			"$ne": id,
		},
		"name": name,
	}
	if uri != "" {
		filter = bson.M{
			"_id": bson.M{
				"$ne": id,
			},
			"$or": []bson.M{
				{"name": name},
				{"uri": uri, "method": method},
			},
		}
	}

	singleRes = l.svcCtx.ApiModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Api
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复api:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == name:
			resp.Msg = "api名称已占用"
		default:
			resp.Msg = fmt.Sprintf("api[%s][%s]已占用", method, uri)
		}

		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //api未占用
	default:
		fmt.Printf("[Error]查询Api：%s\n", singleRes.Err().Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//4.api更新
	var update = bson.M{
		"$set": bson.M{
			"type":       req.Type,
			"sort_id":    req.SortId,
			"parent_id":  req.ParentId,
			"uri":        req.Uri,
			"method":     req.Method,
			"name":       req.Name,
			"remark":     req.Remark,
			"updated_at": time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.ApiModel.UpdateByID(l.ctx, id, update)
	if err != nil {
		fmt.Printf("[Error]更新Api[%s]：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
