package outbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var errOutboundMutationRejected = errors.New("outbound mutation rejected")

// runOutboundMutation keeps the order and its inventory/material changes in one
// transaction, including when a business validation rejects a partial mutation.
func runOutboundMutation(ctx context.Context, svcCtx *svc.ServiceContext, apply func(context.Context) (*types.BaseResponse, error)) (*types.BaseResponse, error) {
	session, err := svcCtx.DBClient.StartSession()
	if err != nil {
		logx.WithContext(ctx).Errorf("start outbound mutation: %v", err)
		return &types.BaseResponse{Code: http.StatusInternalServerError, Msg: "创建出库事务失败"}, nil
	}
	defer session.EndSession(ctx)
	var response *types.BaseResponse
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var applyErr error
		response, applyErr = apply(sc)
		if applyErr != nil {
			return nil, applyErr
		}
		if response == nil || response.Code != http.StatusOK {
			return nil, errOutboundMutationRejected
		}
		return nil, nil
	})
	if errors.Is(err, errOutboundMutationRejected) && response != nil {
		return response, nil
	}
	if err != nil {
		logx.WithContext(ctx).Errorf("commit outbound mutation: %v", err)
		return &types.BaseResponse{Code: http.StatusInternalServerError, Msg: "出库操作未确认完成，请刷新单据后重试"}, nil
	}
	return response, nil
}

func outboundStateFilter(order model.OutboundOrder) bson.M {
	filter := outboundRevisionFilter(order.Id, order.RevisionNo)
	filter["status"] = order.Status
	return filter
}
