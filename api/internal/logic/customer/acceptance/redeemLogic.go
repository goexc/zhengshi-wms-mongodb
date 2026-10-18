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

type RedeemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 确认承兑兑付到账
func NewRedeemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RedeemLogic {
	return &RedeemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RedeemLogic) Redeem(req *types.CustomerAcceptanceRedeemRequest) (resp *types.CustomerAcceptanceResponse, err error) {
	return documents.RedeemAcceptance(l.ctx, l.svcCtx, req), nil
}
