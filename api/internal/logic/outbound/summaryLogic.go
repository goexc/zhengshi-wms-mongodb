package outbound

import (
	"api/model"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SummaryLogic {
	return &SummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SummaryLogic) Summary(req *types.OutboundSummaryRequest) (resp *types.OutboundSummaryResponse, err error) {
	resp = new(types.OutboundSummaryResponse)

	//1.确定起止时间
	year, month, day := time.Unix(req.StartDate, 0).Date()
	startDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local).Unix()
	year, month, day = time.Unix(req.EndDate, 0).Date()
	endDate := time.Date(year, month, day+1, 0, 0, -1, 0, time.Local).Unix()

	//2.客户是否存在
	customerId, _ := primitive.ObjectIDFromHex(req.CustomerId)
	var customer model.Customer
	err = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId}).Decode(&customer)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户不存在"

			return resp, nil
		}

		fmt.Printf("[Error]查询客户[%s]:%s\n", req.CustomerId, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.查询出库单
	cur, err := l.svcCtx.OutboundOrderModel.Find(l.ctx, bson.M{"customer_id": req.CustomerId, "receipt_time": bson.M{"$gte": startDate, "$lte": endDate}})
	if err != nil {
		fmt.Printf("[Error]查询客户[%s]出库单列表:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var orders = make([]model.OutboundOrder, 0)
	if err = cur.All(l.ctx, &orders); err != nil {
		fmt.Printf("[Error]解析客户[%s]出库单列表:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var orderMap = make(map[string]model.OutboundOrder)
	var ordersCode = make([]string, 0)
	for _, one := range orders {
		orderMap[one.Code] = one
		ordersCode = append(ordersCode, one.Code)
	}

	//4.查询出库单物料列表
	cur, err = l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": bson.M{"$in": ordersCode}})
	if err != nil {
		fmt.Printf("[Error]查询客户[%s]出库单物料列表:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials = make([]model.OutboundOrderMaterial, 0)
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析客户[%s]出库单物料列表:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	for _, material := range materials {
		fmt.Printf("出库单[%s]物料：%s(%s)\n", material.OrderCode, material.OrderCode, material.Name)
		resp.Data = append(resp.Data, types.OutboundOrderRecord{
			Code:          material.OrderCode,
			ReceiptDate:   orderMap[material.OrderCode].ReceiptTime,
			DepartureDate: orderMap[material.OrderCode].DepartureTime,
			OutboundOrderMaterial: types.OutboundOrderMaterial{
				Id:            material.Id.Hex(),
				OrderCode:     material.OrderCode,
				MaterialId:    material.MaterialId,
				Index:         material.Index,
				Price:         material.Price,
				Name:          material.Name,
				Model:         material.Model,
				Specification: material.Specification,
				Quantity:      material.Quantity,
				Unit:          material.Unit,
				Weight:        material.Weight,
			},
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
