package role

import (
	"context"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MenuDistributeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMenuDistributeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MenuDistributeLogic {
	return &MenuDistributeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MenuDistributeLogic) MenuDistribute(req *types.RoleMenusRequest) (resp *types.BaseResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
