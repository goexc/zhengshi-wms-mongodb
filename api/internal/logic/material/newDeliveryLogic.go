package material

import (
	"api/model"
	"context"
	"fmt"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NewDeliveryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNewDeliveryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NewDeliveryLogic {
	return &NewDeliveryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NewDeliveryLogic) NewDelivery(req *types.NewCustomerMaterialRequest) (resp *types.NewCustomerMaterialResponse, err error) {
	resp = new(types.NewCustomerMaterialResponse)
	if req.EndTime < req.StartTime {
		resp.Code = http.StatusBadRequest
		resp.Msg = "结束时间不能早于开始时间"
		return resp, nil
	}

	filter := bson.M{
		"customer_id": req.CustomerId,
		"first_delivery_time": bson.M{
			"$gte": req.StartTime,
			"$lte": req.EndTime,
		},
	}
	if strings.TrimSpace(req.QuoteStatus) != "" {
		filter["quote_status"] = strings.TrimSpace(req.QuoteStatus)
	}
	if strings.TrimSpace(req.MaterialName) != "" {
		filter["material_name"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialName), Options: "i"}
	}
	if strings.TrimSpace(req.MaterialModel) != "" {
		filter["material_model"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialModel), Options: "i"}
	}

	total, err := l.svcCtx.CustomerMaterialDeliveryModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询客户新增物料数量失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "first_delivery_time", Value: -1}, {Key: "created_at", Value: -1}}).
		SetSkip((req.Page - 1) * req.Size).
		SetLimit(req.Size)
	cur, err := l.svcCtx.CustomerMaterialDeliveryModel.Find(l.ctx, filter, opts)
	if err != nil {
		fmt.Printf("[Error]查询客户新增物料失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var deliveries []model.CustomerMaterialDelivery
	if err = cur.All(l.ctx, &deliveries); err != nil {
		fmt.Printf("[Error]解析客户新增物料失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Total = total
	for _, one := range deliveries {
		resp.Data.List = append(resp.Data.List, types.NewCustomerMaterialItem{
			Id:                     one.Id.Hex(),
			CustomerId:             one.CustomerId,
			CustomerName:           one.CustomerName,
			MaterialId:             one.MaterialId,
			MaterialName:           one.MaterialName,
			MaterialModel:          one.MaterialModel,
			MaterialSpecification:  one.MaterialSpecification,
			MaterialUnit:           one.MaterialUnit,
			FirstDeliveryTime:      one.FirstDeliveryTime,
			FirstDeliveryOrderCode: one.FirstDeliveryOrderCode,
			FirstDeliveryQuantity:  one.FirstDeliveryQuantity,
			FirstDeliveryPrice:     one.FirstDeliveryPrice,
			QuoteStatus:            one.QuoteStatus,
			LatestQuoteId:          one.LatestQuoteId,
			LatestQuoteNo:          one.LatestQuoteNo,
			LatestPrice:            one.LatestPrice,
		})
	}
	return resp, nil
}
