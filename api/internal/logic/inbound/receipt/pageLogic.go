package receipt

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
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

func (l *PageLogic) Page(req *types.InboundReceiptsRequest) (resp *types.InboundReceiptsResponse, err error) {
	resp = new(types.InboundReceiptsResponse)

	//1.筛选
	var filter = bson.M{}
	if strings.TrimSpace(req.Code) != "" {
		fmt.Println("查询入库单编号：", req.Code)
		filter["code"] = strings.TrimSpace(req.Code)
	}

	if strings.TrimSpace(req.Status) != "" {
		//filter["status"] = code.InboundReceiptStatusCode(req.Status)
		filter["status"] = req.Status
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

	var opt = options.Find().SetSort(bson.M{"created_at": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)

	cur, err := l.svcCtx.InboundReceiptModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询入库单分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var receipts = make([]model.InboundReceipt, 0)
	if err = cur.All(l.ctx, &receipts); err != nil {
		fmt.Println("[Error]解析入库单分页：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var list = make([]types.InboundReceipt, 0)
	for _, one := range receipts {
		var receipt = types.InboundReceipt{
			Id:            one.Id.Hex(),
			Code:          one.Code,
			Status:        one.Status,
			Type:          one.Type,
			SupplierId:    one.SupplierId,
			SupplierName:  one.SupplierName,
			CustomerId:    one.CustomerId,
			CustomerName:  one.CustomerName,
			ReceivingDate: one.ReceivingDate,
			TotalAmount:   one.TotalAmount,
			Annex:         one.Annex,
			Remark:        one.Remark,
		}

		for _, material := range one.Materials {
			receipt.Materials = append(receipt.Materials, types.InboundMaterialItem{
				Index:             material.Index,
				Id:                material.Id,
				Name:              material.Name,
				Status:            material.Status,
				Model:             material.Model,
				Unit:              material.Unit,
				Price:             material.Price,
				EstimatedQuantity: material.EstimatedQuantity,
				ActualQuantity:    material.ActualQuantity,
				//WarehouseId:       material.WarehouseId,
				//WarehouseZoneId:   material.WarehouseZoneId,
				//WarehouseRackId:   material.WarehouseRackId,
				//WarehouseBinId:    material.WarehouseBinId,
				//WarehouseName:     material.WarehouseName,
				//WarehouseZoneName: material.WarehouseZoneName,
				//WarehouseRackName: material.WarehouseRackName,
				//WarehouseBinName:  material.WarehouseBinName,
			})
		}

		list = append(list, receipt)
	}

	resp.Data.List = list

	resp.Data.Total, err = l.svcCtx.InboundReceiptModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询入库单数量:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
