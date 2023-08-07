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

func (l *AddLogic) Add(req *types.MaterialRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.物料编号是否占用，物料名称是否占用
	var filter = bson.M{
		"$or": []bson.M{
			{"model": strings.TrimSpace(req.Model)},
			{"name": strings.TrimSpace(req.Name)},
		},
	}

	singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Material
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复物料:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Model == strings.TrimSpace(req.Model):
			resp.Msg = "物料型号已占用"
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "物料名称已占用"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //物料标号、名称未占用
	default:
		fmt.Printf("[Error]查询重复物料:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
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

	// 3.分类是否存在
	categoryId, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.CategoryId))
	if err != nil {
		fmt.Printf("[Error]物料分类[%s]id转换：%s\n", req.CategoryId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择物料分类"
		return resp, nil
	}

	var category model.MaterialCategory
	singleRes = l.svcCtx.MaterialCategoryModel.FindOne(l.ctx, bson.M{"_id": categoryId})
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&category); err != nil {
			fmt.Printf("[Error]解析物料分类[%s]:%s\n", req.CategoryId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if category.Status != "启用" {
			resp.Msg = fmt.Sprintf("当前分类已%s", category.Status)
			resp.Code = http.StatusBadRequest
			return resp, nil
		}

	case mongo.ErrNoDocuments: //物料分类未占用
	default:
		fmt.Printf("[Error]查询物料分类:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.物料信息入库
	var m = model.Material{
		Name:             strings.TrimSpace(req.Name),
		CategoryId:       strings.TrimSpace(req.CategoryId),
		CategoryName:     category.Name,
		Image:            strings.TrimSpace(req.Image),
		Model:            strings.TrimSpace(req.Model),
		Material:         strings.TrimSpace(req.Material),
		Specification:    strings.TrimSpace(req.Specification),
		SurfaceTreatment: strings.TrimSpace(req.SurfaceTreatment),
		StrengthGrade:    strings.TrimSpace(req.StrengthGrade),
		Quantity:         req.Quantity,
		Unit:             strings.TrimSpace(req.Unit),
		Remark:           strings.TrimSpace(req.Remark),
		Creator:          userId,
		CreatorName:      l.ctx.Value("name").(string),
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	_, err = l.svcCtx.MaterialModel.InsertOne(l.ctx, &m)
	if err != nil {
		fmt.Printf("[Error]物料入库:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
