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

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 物料信息
func NewInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InfoLogic {
	return &InfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InfoLogic) Info(req *types.MaterialIdRequest) (resp *types.MaterialResponse, err error) {
	resp = new(types.MaterialResponse)

	//1.参数校验
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]物料[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	fmt.Println("物料id：", req.Id)

	//2.物料id是否存在
	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, filter)

	var material model.Material

	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&material); err != nil {
			fmt.Printf("[Error]解析物料[%s]信息:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //物料id不存在
		resp.Code = http.StatusInternalServerError
		resp.Msg = "物料不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询物料[%s]:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil

	}

	var info = types.Material{
		Id:               req.Id,
		Image:            material.Image,
		CategoryId:       material.CategoryId,
		CategoryName:     material.CategoryName,
		Name:             material.Name,
		Material:         material.Material,
		Specification:    material.Specification,
		Model:            material.Model,
		SurfaceTreatment: material.SurfaceTreatment,
		StrengthGrade:    material.StrengthGrade,
		Quantity:         material.Quantity,
		Unit:             material.Unit,
		Remark:           material.Remark,
		Creator:          material.Creator.Hex(),
		CreatorName:      material.CreatorName,
		CreatedAt:        material.CreatedAt,
		UpdatedAt:        material.UpdatedAt,
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = info
	return resp, nil
}
