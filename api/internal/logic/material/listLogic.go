package material

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *types.MaterialsRequest) (resp *types.MaterialsResponse, err error) {
	resp = new(types.MaterialsResponse)

	//1.构造过滤条件
	var filter = bson.M{}
	var option = options.Find().SetSort(bson.M{"created_at": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)
	name := strings.TrimSpace(req.Name)
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter["name"] = regex
	}
	if strings.TrimSpace(req.Code) != "" {
		filter["code"] = strings.TrimSpace(req.Code)
	}

	if strings.TrimSpace(req.Material) != "" {
		filter["material"] = strings.TrimSpace(req.Material)
	}

	if strings.TrimSpace(req.Specification) != "" {
		filter["specification"] = strings.TrimSpace(req.Specification)
	}

	if strings.TrimSpace(req.Model) != "" {
		filter["model"] = strings.TrimSpace(req.Model)
	}

	if strings.TrimSpace(req.SurfaceTreatment) != "" {
		filter["surface_treatment"] = strings.TrimSpace(req.SurfaceTreatment)
	}

	if strings.TrimSpace(req.StrengthGrade) != "" {
		filter["strength_grade"] = strings.TrimSpace(req.StrengthGrade)
	}

	fmt.Println("过滤条件：", filter)

	//2.查询分页
	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Printf("[Error]物料分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials []model.Material
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析原材料分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	fmt.Println(materials)

	//3.统计总数
	total, err := l.svcCtx.MaterialModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]物料总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.Total = total
	resp.Data.List = make([]types.Material, 0)
	for _, m := range materials {
		resp.Data.List = append(resp.Data.List, types.Material{
			Id:               m.Id.Hex(),
			Code:             m.Code,
			Name:             m.Name,
			Material:         m.Material,
			Specification:    m.Specification,
			Model:            m.Model,
			SurfaceTreatment: m.SurfaceTreatment,
			StrengthGrade:    m.StrengthGrade,
			Unit:             m.Unit,
			Remark:           m.Remark,
			CreatedAt:        m.CreatedAt,
			UpdatedAt:        m.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
