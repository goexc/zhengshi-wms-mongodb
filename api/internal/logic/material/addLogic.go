package material

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
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
			{"code": strings.TrimSpace(req.Code)},
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

	//2.物料信息入库
	var m = model.Material{
		Code:             strings.TrimSpace(req.Code),
		Name:             strings.TrimSpace(req.Name),
		Material:         strings.TrimSpace(req.Material),
		Specification:    strings.TrimSpace(req.Specification),
		Model:            strings.TrimSpace(req.Model),
		SurfaceTreatment: strings.TrimSpace(req.SurfaceTreatment),
		StrengthGrade:    strings.TrimSpace(req.StrengthGrade),
		Unit:             strings.TrimSpace(req.Unit),
		Remark:           strings.TrimSpace(req.Remark),
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	_, err = l.svcCtx.MaterialModel.InsertOne(l.ctx, &m)
	if err != nil {
		fmt.Printf("[Error]物料入库:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
