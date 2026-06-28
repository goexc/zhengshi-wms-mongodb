package quote

import (
	"context"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportLogic {
	return &ExportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExportLogic) Export(req *types.MaterialQuoteIdRequest) (fileName string, body []byte, resp *types.BaseResponse, err error) {
	return (&quoteService{ctx: l.ctx, svcCtx: l.svcCtx}).export(req)
}
