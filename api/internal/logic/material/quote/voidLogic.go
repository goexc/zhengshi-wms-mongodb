package quote

import (
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

func NewVoidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VoidLogic {
	return &VoidLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VoidLogic) Void(req *types.MaterialQuoteIdRequest) (resp *types.BaseResponse, err error) {
	return (&quoteService{ctx: l.ctx, svcCtx: l.svcCtx}).void(req)
}
