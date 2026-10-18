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

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 销售开票分页
func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.CustomerDocumentPageRequest) (resp *types.CustomerInvoicePageResponse, err error) {
	return documents.PageInvoice(l.ctx, l.svcCtx, req), nil
}
