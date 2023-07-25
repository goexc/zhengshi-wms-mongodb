package material

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

type AddCategoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCategoryLogic {
	return &AddCategoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddCategoryLogic) AddCategory(req *types.MaterialCategoryRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	var filter = bson.M{}

	//1.当前物料分类是否存在
	//i 表示不区分大小写
	filter = bson.M{"name": strings.TrimSpace(req.Name)}

	count, err := l.svcCtx.MaterialCategoryModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询物料分类[%s]是否占用:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "物料分类名称已占用"
		return resp, nil
	}

	// 2.账号是否存在
	uid := l.ctx.Value("uid").(string)
	userId, err := primitive.ObjectIDFromHex(strings.TrimSpace(uid))
	if err != nil {
		fmt.Printf("[Error]用户[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//3.信息入库
	var c = model.MaterialCategory{
		ParentId:    strings.TrimSpace(req.ParentId),
		SortId:      req.SortId,
		Name:        strings.TrimSpace(req.Name),
		Status:      req.Status,
		Remark:      strings.TrimSpace(req.Remark),
		Creator:     userId,
		CreatorName: l.ctx.Value("name").(string),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	_, err = l.svcCtx.MaterialCategoryModel.InsertOne(l.ctx, &c)
	if err != nil {
		fmt.Printf("[Error]物料分类入库:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
