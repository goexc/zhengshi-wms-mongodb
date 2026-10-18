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

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改销售开票
func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.CustomerInvoiceSaveRequest) (resp *types.CustomerInvoiceResponse, err error) {
	return documents.SaveInvoice(l.ctx, l.svcCtx, req, false), nil
}
