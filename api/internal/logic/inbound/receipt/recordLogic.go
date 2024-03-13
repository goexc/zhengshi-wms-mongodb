package receipt

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordLogic {
	return &RecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecordLogic) Record(req *types.InboundReceivedRecordsRequest) (resp *types.InboundReceivedRecordsResponse, err error) {
	resp = new(types.InboundReceivedRecordsResponse)

	//1.入库单是否存在
	receiptId, _ := primitive.ObjectIDFromHex(req.InboundReceiptId)
	var receipt model.InboundReceipt

	//1.入库单号是否存在
	var filter = bson.M{"_id": receiptId}
	singleRes := l.svcCtx.InboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析入库单:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //入库单未占用
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询入库单:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.查询批次入库记录
	filter = bson.M{"inbound_receipt_id": req.InboundReceiptId}
	opts := options.Find().SetSort(bson.M{"receiving_date": 1})
	cur, err := l.svcCtx.InboundReceiptReceiveModel.Find(l.ctx, filter, opts)
	if err != nil {
		fmt.Printf("[Error]查询批次入库记录:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var records = make([]model.InboundReceive, 0)
	if err = cur.All(l.ctx, &records); err != nil {
		fmt.Println("[Error]解析批次入库记录：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	fmt.Println("批次入库记录数量：", len(records))

	var list = make([]types.InboundReceivedRecord, 0)
	for _, record := range records {
		var materials = make([]types.InboundReceiveMaterial, 0)
		for _, material := range record.Materials {
			materials = append(materials, types.InboundReceiveMaterial{
				Id:                material.Id,
				Index:             material.Index,
				Price:             material.Price,
				Name:              material.Name,
				Model:             material.Model,
				Unit:              material.Unit,
				ActualQuantity:    material.ActualQuantity,
				Status:            material.Status,
				WarehouseName:     material.WarehouseName,
				WarehouseZoneName: material.WarehouseZoneName,
				WarehouseRackName: material.WarehouseRackName,
				WarehouseBinName:  material.WarehouseBinName,
			})
		}

		list = append(list, types.InboundReceivedRecord{
			Id:               record.Id.Hex(),
			Code:             record.Code,
			InboundReceiptId: record.InboundReceiptId,
			CarrierName:      record.CarrierName,
			CarrierCost:      record.CarrierCost,
			OtherCost:        record.OtherCost,
			TotalAmount:      record.TotalAmount,
			ReceivingDate:    record.ReceivingDate,
			Materials:        materials,
			Annex:            record.Annex,
			Remark:           record.Remark,
			CreatorId:        record.CreatorId,
			CreatorName:      record.CreatorName,
			CreatedAt:        record.CreatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = list

	return resp, nil
}
