package material

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

type UpdateCategoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCategoryLogic {
	return &UpdateCategoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCategoryLogic) UpdateCategory(req *types.MaterialCategoryRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]物料分类[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//1.物料分类名称是否占用
	var filter = bson.M{
		"name": strings.TrimSpace(req.Name),
		"_id":  bson.M{"$ne": id},
	}
	singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Material
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复物料分类:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Msg = "物料分类名称已占用"
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //物料分类标号、名称未占用
	default:
		fmt.Printf("[Error]查询重复物料分类:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.更新物料分类信息
	update := bson.M{
		"$set": bson.M{
			"parent_id":  strings.TrimSpace(req.ParentId),
			"sort_id":    req.SortId,
			"name":       strings.TrimSpace(req.Name),
			"status":     strings.TrimSpace(req.Status),
			"remark":     strings.TrimSpace(req.Remark),
			"updated_at": time.Now().Unix(),
		},
	}
	_, err = l.svcCtx.MaterialCategoryModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新物料分类[%s]:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
