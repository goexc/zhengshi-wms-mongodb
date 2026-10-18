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

type VoidLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 作废客户承兑
func NewVoidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VoidLogic {
	return &VoidLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VoidLogic) Void(req *types.CustomerDocumentVoidRequest) (resp *types.CustomerAcceptanceResponse, err error) {
	return documents.VoidAcceptance(l.ctx, l.svcCtx, req), nil
}
