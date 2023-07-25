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

	if strings.TrimSpace(req.CategoryId) != "" {
		//查询分类及其子类
		categorys := l.getSubCategorys(req.CategoryId)
		categorys = append(categorys, req.CategoryId)
		fmt.Println("分类及其子类：", categorys)

		filter["category_id"] = bson.M{"$in": categorys}
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
			Name:             m.Name,
			Image:            m.Image,
			CategoryId:       m.CategoryId,
			CategoryName:     m.CategoryName,
			Material:         m.Material,
			Specification:    m.Specification,
			Model:            m.Model,
			SurfaceTreatment: m.SurfaceTreatment,
			StrengthGrade:    m.StrengthGrade,
			Unit:             m.Unit,
			Remark:           m.Remark,
			Creator:          m.Creator.Hex(),
			CreatorName:      m.CreatorName,
			CreatedAt:        m.CreatedAt,
			UpdatedAt:        m.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}

// 查询物料分类的子类
func (l *ListLogic) getSubCategorys(parentId string) (subIds []string) {
	cur, err := l.svcCtx.MaterialCategoryModel.Find(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询物料分类:%s\n", err.Error())
		return nil
	}

	defer cur.Close(l.ctx)

	var categorys []model.MaterialCategory
	if err = cur.All(l.ctx, &categorys); err != nil {
		fmt.Println("[Error]解析物料分类列表：", err.Error())
		return nil
	}

	if len(categorys) == 0 {
		return
	}

	//2.物料分类分组
	/*var list = make([]types.MaterialCategory, 0)
	for _, one := range categorys {
		list = append(list, types.MaterialCategory{
			Id:          one.Id.Hex(),
			ParentId:    one.ParentId,
			SortId:      one.SortId,
			Name:        one.Name,
			Status:      one.Status,
			Remark:      one.Remark,
			CreatorName: one.CreatorName,
			CreatedAt:   one.CreatedAt,
			UpdatedAt:   one.UpdatedAt,
		})
	}*/

	/*//3.构造树形数据结构
	categoryMap := make(map[string]*types.MaterialCategory)

	//遍历 categorys 切片，将每个 Category 添加到 map 中
	for i := range list {
		categoryMap[list[i].Id] = &list[i]
	}

	//遍历 list 切片，构建树形结构
	var rootCategory = make([]*types.MaterialCategory, 0)
	for i := range list {
		if parent, ok := categoryMap[list[i].ParentId]; ok {
			parent.Children = append(parent.Children, &list[i])
		} else {
			rootCategory = append(rootCategory, &list[i])
		}
	}

	//取出当前分类的子类
	parent, ok := categoryMap[parentId]
	if !ok {
		return
	}

	subCategorys := parent.Children*/

	var dfs func(string)
	dfs = func(id string) {
		for _, one := range categorys {
			if one.ParentId == id {
				subIds = append(subIds, one.Id.Hex())
				dfs(one.Id.Hex())
			}
		}
	}

	dfs(parentId)

	return
}
