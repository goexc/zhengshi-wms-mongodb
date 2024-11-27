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

func (l *PageLogic) Page(req *types.OutboundOrdersRequest) (resp *types.OutboundOrdersResponse, err error) {
	resp = new(types.OutboundOrdersResponse)
	fmt.Printf("请求参数:%#v\n", req)
	defer func() {
		fmt.Printf("出库单列表：:%#v\n", resp.Data.List)
		fmt.Printf("出库单分页数量：:%d\n", len(resp.Data.List))
	}()

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
	for _, one := range receipts {
		var receipt = types.OutboundOrderItem{
			Id:            one.Id.Hex(),
			Code:          one.Code,
			Status:        one.Status,
			IsPack:        one.IsPack,
			IsWeigh:       one.IsWeigh,
			Type:          one.Type,
			SupplierId:    one.SupplierId,
			SupplierName:  one.SupplierName,
			CustomerId:    one.CustomerId,
			CustomerName:  one.CustomerName,
			CarrierId:     one.CarrierId,
			CarrierName:   one.CarrierName,
			CarrierCost:   one.CarrierCost,
			OtherCost:     one.OtherCost,
			TotalAmount:   one.TotalAmount,
			Annex:         strings.Split(one.Annex, ","),
			Remark:        one.Remark,
			ConfirmTime:   one.ConfirmTime,
			PickingTime:   one.PickingTime,
			PackingTime:   one.PackingTime,
			WeighingTime:  one.WeighingTime,
			DepartureTime: one.DepartureTime,
			ReceiptTime:   one.ReceiptTime,
			Materials:     make([]types.OutboundOrderMaterial, 0),
		}

		list = append(list, receipt)
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

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
