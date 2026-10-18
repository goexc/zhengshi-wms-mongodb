// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	"context"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RemovePreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 出库单删除预览
func NewRemovePreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemovePreviewLogic {
	return &RemovePreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemovePreviewLogic) RemovePreview(req *types.OutboundOrderIdRequest) (resp *types.OutboundRemovePreviewResponse, err error) {
	resp = &types.OutboundRemovePreviewResponse{}
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil || id.IsZero() {
		resp.Code, resp.Msg = http.StatusBadRequest, "出库单ID格式错误"
		return resp, nil
	}
	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		l.Errorf("start outbound removal preview session: %v", err)
		resp.Code, resp.Msg = http.StatusInternalServerError, "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)
	var plan *removalPlan
	_, err = session.WithTransaction(l.ctx, func(dbCtx mongo.SessionContext) (interface{}, error) {
		var e error
		plan, e = readRemovalPlan(dbCtx, l.svcCtx, id)
		return nil, e
	}, removalTransactionOptions())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Code, resp.Msg = http.StatusNotFound, "出库单已删除或不存在"
			return resp, nil
		}
		if message, ok := customerfinance.IsBusinessError(err); ok && plan != nil {
			plan.Preview.Allowed = false
			plan.Preview.BlockReason = message
			plan.Preview.PreviewToken = ""
			resp.Code, resp.Msg, resp.Data = http.StatusOK, "成功", plan.Preview
			return resp, nil
		}
		l.Errorf("preview outbound removal %s: %v", id.Hex(), err)
		resp.Code, resp.Msg = http.StatusInternalServerError, "服务器内部错误"
		return resp, nil
	}
	resp.Code, resp.Msg, resp.Data = http.StatusOK, "成功", plan.Preview
	return resp, nil
}
