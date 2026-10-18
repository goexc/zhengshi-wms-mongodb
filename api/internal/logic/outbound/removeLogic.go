package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RemoveLogic) Remove(req *types.OutboundOrderRemoveRequest) (*types.BaseResponse, error) {
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil || id.IsZero() {
		return &types.BaseResponse{Code: http.StatusBadRequest, Msg: "出库单ID格式错误"}, nil
	}
	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		l.Errorf("start outbound removal session: %v", err)
		return &types.BaseResponse{Code: http.StatusInternalServerError, Msg: "服务器内部错误"}, nil
	}
	defer session.EndSession(l.ctx)
	alreadyAbsent := false
	var committedPlan *removalPlan
	// WithTransaction retries the full callback on write conflicts and retries
	// an uncertain commit without repeating stock or finance side effects.
	_, err = session.WithTransaction(l.ctx, func(dbCtx mongo.SessionContext) (interface{}, error) {
		alreadyAbsent = false
		committedPlan = nil
		plan, e := readRemovalPlan(dbCtx, l.svcCtx, id)
		if e == mongo.ErrNoDocuments {
			alreadyAbsent = true
			return nil, nil
		}
		if e != nil {
			return nil, e
		}
		token := strings.TrimSpace(req.PreviewToken)
		if token == "" && plan.Snapshot.Order.Status == "已签收" {
			return nil, removalError("已签收出库单请先查看删除预览再确认删除")
		}
		if token != "" && token != plan.Preview.PreviewToken {
			return nil, errRemovalPreviewChanged
		}
		if e = applyRemovalPlan(dbCtx, l.svcCtx, plan); e != nil {
			return nil, e
		}
		committedPlan = plan
		return nil, nil
	}, removalTransactionOptions())
	if err != nil {
		if errors.Is(err, errRemovalPreviewChanged) {
			return &types.BaseResponse{Code: http.StatusConflict, Msg: errRemovalPreviewChanged.Error()}, nil
		}
		if message, ok := customerfinance.IsBusinessError(err); ok {
			return &types.BaseResponse{Code: http.StatusBadRequest, Msg: message}, nil
		}
		l.Errorf("remove outbound %s: %v", id.Hex(), err)
		return &types.BaseResponse{Code: http.StatusInternalServerError, Msg: "删除失败或结果暂未确认，请刷新列表后重试"}, nil
	}
	message := "成功"
	if alreadyAbsent {
		message = "出库单已删除或不存在"
	} else if committedPlan != nil {
		batches := make([]map[string]interface{}, 0, len(committedPlan.Changes))
		for _, change := range committedPlan.Changes {
			batches = append(batches, map[string]interface{}{"inventory_id": change.Inventory.Id.Hex(), "unit": change.Inventory.Unit, "release": change.Release, "restore": change.Restore})
		}
		l.Infow("outbound_removed", logx.Field("uid", l.ctx.Value("uid")), logx.Field("order_id", id.Hex()), logx.Field("code", committedPlan.Snapshot.Order.Code), logx.Field("deleted_transactions", len(committedPlan.TransactionIDs)), logx.Field("inventory_batch_count", len(batches)), logx.Field("inventory_batches", batches))
	}
	return &types.BaseResponse{Code: http.StatusOK, Msg: message}, nil
}
