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

func (l *UpdateLogic) Update(req *types.MaterialRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]物料[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//1.物料编号是否占用，物料名称是否占用
	var filter = bson.M{
		"$or": []bson.M{
			{"model": strings.TrimSpace(req.Model)},
			{"name": strings.TrimSpace(req.Name)},
		},
		"_id": bson.M{"$ne": id},
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

	// 3.分类是否存在
	fmt.Println("物料分类id:", req.CategoryId)
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

	//2.更新物料信息
	update := bson.M{
		"$set": bson.M{
			"name":              strings.TrimSpace(req.Name),
			"category_id":       strings.TrimSpace(req.CategoryId),
			"category_name":     category.Name,
			"image":             strings.TrimSpace(req.Image),
			"model":             strings.TrimSpace(req.Model),
			"material":          strings.TrimSpace(req.Material),
			"specification":     strings.TrimSpace(req.Specification),
			"surface_treatment": strings.TrimSpace(req.SurfaceTreatment),
			"strength_grade":    strings.TrimSpace(req.StrengthGrade),
			"quantity":          req.Quantity,
			"unit":              strings.TrimSpace(req.Unit),
			"remark":            strings.TrimSpace(req.Remark),
			"updated_at":        time.Now().Unix(),
		},
	}
	_, err = l.svcCtx.MaterialModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新物料[%s]:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
