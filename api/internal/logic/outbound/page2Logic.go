package outbound

import (
	"api/model"
	"api/pkg/code"
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

type Page2Logic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPage2Logic(ctx context.Context, svcCtx *svc.ServiceContext) *Page2Logic {
	return &Page2Logic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *Page2Logic) Page2(req *types.OutboundOrdersRequest) (resp *types.OutboundOrdersResponse, err error) {
	resp = new(types.OutboundOrdersResponse)

	//0.参数校验
	if req.StartTime > 0 && req.EndTime > 0 && req.StartTime > req.EndTime {
		resp.Code = http.StatusBadRequest
		resp.Msg = "签收起始时间不能晚于截止时间"
		return resp, nil
	}

	//1.筛选
	var filter = bson.M{}
	if strings.TrimSpace(req.Code) != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + strings.TrimSpace(req.Code) + ".*", Options: "i"}}
		filter["code"] = regex
	}

	if !(code.OutboundStatusRange(req.Status) == nil) {
		filter["status"] = bson.M{"$in": code.OutboundStatusRange(req.Status)}
	}

	if req.IsPack != -1 {
		filter["is_pack"] = req.IsPack
	}

	if req.IsWeigh != -1 {
		filter["is_weigh"] = req.IsWeigh
	}

	if strings.TrimSpace(req.Type) != "" {
		filter["type"] = strings.TrimSpace(req.Type)
	}

	if strings.TrimSpace(req.SupplierId) != "" {
		filter["supplier_id"] = strings.TrimSpace(req.SupplierId)
	}

	if strings.TrimSpace(req.CustomerId) != "" {
		filter["customer_id"] = strings.TrimSpace(req.CustomerId)
	}

	switch true {
	case req.StartTime > 0 && req.EndTime > 0:
		filter["receipt_time"] = bson.M{"$gte": req.StartTime, "$lte": req.EndTime}
	case req.StartTime == 0 && req.EndTime > 0:
		filter["receipt_time"] = bson.M{"$lte": req.EndTime}
	case req.StartTime > 0 && req.EndTime == 0:
		filter["receipt_time"] = bson.M{"$gte": req.StartTime}
	default: //忽略
	}

	var opt = options.Find().SetSort(bson.M{"receipt_time": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)

	//2.查询出库单分页
	cur, err := l.svcCtx.OutboundOrderModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询出库单分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var receipts = make([]model.OutboundOrder, 0)
	if err = cur.All(l.ctx, &receipts); err != nil {
		fmt.Println("[Error]解析出库单分页：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var list = make([]types.OutboundOrderItem, 0) //出库单列表
	var codes = make([]string, 0)                 //出库单号
	for _, one := range receipts {
		var receipt = types.OutboundOrderItem{
			Id:           one.Id.Hex(),
			Code:         one.Code,
			Status:       one.Status,
			IsPack:       one.IsPack,
			IsWeigh:      one.IsWeigh,
			Type:         one.Type,
			SupplierId:   one.SupplierId,
			SupplierName: one.SupplierName,
			CustomerId:   one.CustomerId,
			CustomerName: one.CustomerName,
			CarrierId:    one.CarrierId,
			CarrierName:  one.CarrierName,
			CarrierCost:  one.CarrierCost,
			OtherCost:    one.OtherCost,
			TotalAmount:  one.TotalAmount,
			Annex:        nil,
			//Receipt:       nil,
			Remark:        one.Remark,
			ConfirmTime:   one.ConfirmTime,
			PickingTime:   one.PickingTime,
			PackingTime:   one.PackingTime,
			WeighingTime:  one.WeighingTime,
			DepartureTime: one.DepartureTime,
			ReceiptTime:   one.ReceiptTime,
			Materials:     make([]types.OutboundOrderMaterial, 0),
		}

		if len(one.Annex) > 0 {
			receipt.Annex = strings.Split(one.Annex, ",")
		}
		//if len(one.Receipt) > 0 {
		//	receipt.Receipt = strings.Split(one.Receipt, ",")
		//}

		list = append(list, receipt)
		codes = append(codes, one.Code)
	}

	resp.Data.List = list

	//3.查询出库单数量
	resp.Data.Total, err = l.svcCtx.OutboundOrderModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询出库单数量:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.查询物料列表
	if len(resp.Data.List) == 0 {
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	cur, err = l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": bson.M{"$in": codes}})
	if err != nil {
		fmt.Printf("[Error]查询出库单分页物料:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials = make([]model.OutboundOrderMaterial, 0)
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单分页物料：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var receiptMaterials = make(map[string][]types.OutboundOrderMaterial)
	for _, one := range materials {
		//if _, ok := receiptMaterials[one.OrderCode]; !ok {
		receiptMaterials[one.OrderCode] = append(receiptMaterials[one.OrderCode], types.OutboundOrderMaterial{
			Id:            one.Id.Hex(),
			OrderCode:     one.OrderCode,
			MaterialId:    one.MaterialId,
			Index:         one.Index,
			Price:         one.Price,
			Name:          one.Name,
			Model:         one.Model,
			Specification: one.Specification,
			Quantity:      one.Quantity,
			Unit:          one.Unit,
			Weight:        one.Weight,
		})
		//}
	}

	for idx, one := range list {
		list[idx].Materials = receiptMaterials[one.Code]
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
