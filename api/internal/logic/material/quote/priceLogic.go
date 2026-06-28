package quote

import (
	"context"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PriceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPriceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PriceLogic {
	return &PriceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PriceLogic) Price(req *types.MaterialQuotePriceRequest) (resp *types.MaterialQuoteResponse, err error) {
	return (&quoteService{ctx: l.ctx, svcCtx: l.svcCtx}).price(req)
}
