package outbound

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MaterialsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMaterialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MaterialsLogic {
	return &MaterialsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MaterialsLogic) Materials(req *types.OutboundOrderMaterialsRequest) (resp *types.OutboundOrderMaterialsResponse, err error) {
	resp = new(types.OutboundOrderMaterialsResponse)

	//1.入库单号是否存在
	count, err := l.svcCtx.OutboundOrderModel.CountDocuments(l.ctx, bson.M{"code": req.OrderCode})
	if err != nil {
		fmt.Printf("[Error]出库单号[%s]是否存在：%s\n", req.OrderCode, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	}

	//2.查询物料列表
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": req.OrderCode})
	if err != nil {
		fmt.Printf("[Error]查询入库单[%s]物料:%s\n", req.OrderCode, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials = make([]model.OutboundOrderMaterial, 0)
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析入库单[%s]物料：%s\n", req.OrderCode, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	for _, one := range materials {
		resp.Data = append(resp.Data, types.OutboundOrderMaterial{
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
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
