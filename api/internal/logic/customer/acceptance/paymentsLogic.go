// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package acceptance

import (
	"api/internal/logic/customer/documents"
	"context"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PaymentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 可关联的客户回款
func NewPaymentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PaymentsLogic {
	return &PaymentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PaymentsLogic) Payments(req *types.CustomerAcceptancePaymentsRequest) (resp *types.CustomerAcceptancePaymentsResponse, err error) {
	return documents.Payments(l.ctx, l.svcCtx, req), nil
}
