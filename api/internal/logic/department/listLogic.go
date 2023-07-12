package department

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

func (l *ListLogic) List() (resp *types.DepartmentsResponse, err error) {
	resp = new(types.DepartmentsResponse)

	//1.部门列表查询
	filter := bson.M{}
	option := options.Find().SetSort(bson.M{"sort_id": 1})
	cur, err := l.svcCtx.DepartmentModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Println("[Error]查询部门列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var departments []model.Department
	if err = cur.All(l.ctx, &departments); err != nil {
		fmt.Println("[Error]解析部门列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.部门分组
	var list = make([]types.Department, 0)
	for _, one := range departments {
		list = append(list, types.Department{
			Id:        one.Id.Hex(),
			SortId:    one.SortId,
			ParentId:  one.ParentId,
			Name:      one.Name,
			Code:      one.Code,
			Remark:    one.Remark,
			CreatedAt: one.CreatedAt,
			UpdatedAt: one.UpdatedAt,
		})
	}

	//3.构造树形数据结构
	departmentMap := make(map[string]*types.Department)

	//遍历 departments 切片，将每个 department 添加到 map 中
	for i := range list {
		departmentMap[list[i].Id] = &list[i]
	}

	//遍历 list 切片，构建树形结构
	var rootDepartment = make([]*types.Department, 0)
	for i := range list {
		if parent, ok := departmentMap[list[i].ParentId]; ok {
			parent.Children = append(parent.Children, &list[i])
		} else {
			rootDepartment = append(rootDepartment, &list[i])
		}
	}

	resp.Data = rootDepartment
	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
