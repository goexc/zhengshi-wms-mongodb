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

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 登记客户承兑
func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.CustomerAcceptanceSaveRequest) (resp *types.CustomerAcceptanceResponse, err error) {
	return documents.SaveAcceptance(l.ctx, l.svcCtx, req, true), nil
}
