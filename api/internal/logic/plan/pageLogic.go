package plan

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

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.PlansRequest) (resp *types.PlansResponse, err error) {
	resp = new(types.PlansResponse)

	var filter = bson.M{}
	var option = options.Find().SetSort(bson.M{"deadline": 1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)

	if req.Status != "" {
		filter["status"] = req.Status
	}

	//1.查询分页
	cur, err := l.svcCtx.PlanModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Printf("[Error]计划分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var plans = make([]model.Plan, 0)
	if err = cur.All(l.ctx, &plans); err != nil {
		fmt.Printf("[Error]解析计划分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	for _, one := range plans {
		resp.Data.List = append(resp.Data.List, types.Plan{
			Id:               one.Id.Hex(),
			Type:             one.Type,
			Status:           one.Status,
			CustomerId:       one.CustomerId,
			CustomerName:     one.CustomerName,
			SupplierId:       one.SupplierId,
			SupplierName:     one.SupplierName,
			MaterialId:       one.MaterialId,
			MaterialName:     one.MaterialName,
			MaterialModel:    one.MaterialModel,
			MaterialImage:    one.MaterialImage,
			MaterialUnit:     one.MaterialUnit,
			MaterialQuantity: one.MaterialQuantity,
			Deadline:         one.Deadline,
		})
	}

	//2.统计总数
	resp.Data.Total, err = l.svcCtx.PlanModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询计划总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
