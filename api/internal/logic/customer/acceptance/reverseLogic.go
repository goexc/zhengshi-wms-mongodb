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

type ReverseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销承兑兑付
func NewReverseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReverseLogic {
	return &ReverseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReverseLogic) Reverse(req *types.CustomerDocumentVoidRequest) (resp *types.CustomerAcceptanceResponse, err error) {
	return documents.ReverseAcceptance(l.ctx, l.svcCtx, req), nil
}
