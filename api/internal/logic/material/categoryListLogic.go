package material

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CategoryListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCategoryListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CategoryListLogic {
	return &CategoryListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CategoryListLogic) CategoryList() (resp *types.MaterialCategoryListResponse, err error) {
	resp = new(types.MaterialCategoryListResponse)

	//1.物料分类列表查询
	filter := bson.M{}
	option := options.Find().SetSort(bson.M{"sort_id": 1})
	cur, err := l.svcCtx.MaterialCategoryModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Println("[Error]查询物料分类列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var categorys []model.MaterialCategory
	if err = cur.All(l.ctx, &categorys); err != nil {
		fmt.Println("[Error]解析物料分类列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.物料分类分组
	var list = make([]types.MaterialCategory, 0)
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

	}

	//3.构造树形数据结构
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

	resp.Data = rootCategory
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
