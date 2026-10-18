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

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 客户承兑详情
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.CustomerDocumentDetailRequest) (resp *types.CustomerAcceptanceResponse, err error) {
	return documents.DetailAcceptance(l.ctx, l.svcCtx, req), nil
}
