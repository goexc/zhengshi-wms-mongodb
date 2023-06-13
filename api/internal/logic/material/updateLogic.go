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
			{"code": strings.TrimSpace(req.Code)},
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
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "物料编号已占用"
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

	//2.更新物料信息
	update := bson.M{
		"$set": bson.M{
			"code":              strings.TrimSpace(req.Code),
			"name":              strings.TrimSpace(req.Name),
			"material":          strings.TrimSpace(req.Material),
			"specification":     strings.TrimSpace(req.Specification),
			"model":             strings.TrimSpace(req.Model),
			"surface_treatment": strings.TrimSpace(req.SurfaceTreatment),
			"strength_grade":    strings.TrimSpace(req.StrengthGrade),
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
