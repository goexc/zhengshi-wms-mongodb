package auth

import (
	"api/internal/svc"
	"api/internal/types"
	"context"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout() (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)

	//1.删除token缓存
	if uid != "" {
		err = l.svcCtx.Cache.DelCtx(l.ctx, fmt.Sprintf(userTokenKey, uid))
		if err != nil {
			fmt.Printf("[Error]删除用户[%s]Token缓存:%s\n", uid, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
