package receipt

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 可删除的入库单状态
var canDeleteStatus = map[string]string{"待审核": "", "审核不通过": ""}

func (l *RemoveLogic) Remove(req *types.InboundReceiptIdRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//待审核、审核不通过的入库单可以删除

	//1.入库单是否存在
	id, _ := primitive.ObjectIDFromHex(req.Id)
	filter := bson.M{"_id": id}
	var receipt model.InboundReceipt
	singleRes := l.svcCtx.InboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析重复个人:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if _, ok := canDeleteStatus[code.InboundReceiptStatusText(receipt.Status)]; !ok {
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("无法删除[%s]状态的入库单", code.InboundReceiptStatusText(receipt.Status))
			return resp, nil
		}

	case mongo.ErrNoDocuments: //入库单不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询入库单[%s]是否存在:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.删除入库单
	_, err = l.svcCtx.InboundReceiptModel.DeleteOne(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]删除入库单[%s]:%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
