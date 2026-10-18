// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package invoice

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

// 作废销售开票
func NewVoidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VoidLogic {
	return &VoidLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VoidLogic) Void(req *types.CustomerDocumentVoidRequest) (resp *types.CustomerInvoiceResponse, err error) {
	return documents.VoidInvoice(l.ctx, l.svcCtx, req), nil
}
