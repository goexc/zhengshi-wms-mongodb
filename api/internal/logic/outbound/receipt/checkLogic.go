package receipt

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckLogic {
	return &CheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckLogic) Check(req *types.OutboundReceiptCheckRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	//审核：只有待审核出库单可以审核，审核结果：审核不通过、审核通过

	//1.出库单是否存在
	id, _ := primitive.ObjectIDFromHex(req.Id)
	filter := bson.M{"_id": id}
	var receipt model.OutboundReceipt
	singleRes := l.svcCtx.OutboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析重复个人:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if receipt.Status != code.OutboundReceiptStatusCode("待审核") {
			resp.Code = http.StatusBadRequest
			resp.Msg = "出库单不能重复审核"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //出库单不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询出库单[%s]是否存在:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.审核
	update := bson.M{
		"$set": bson.M{
			"status": code.OutboundReceiptStatusCode(req.Status),
		},
	}

	_, err = l.svcCtx.OutboundReceiptModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]审核出库单[%s]：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"

	return resp, nil
}
