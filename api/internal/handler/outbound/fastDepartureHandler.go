// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package outbound

import (
	"net/http"

	"api/internal/logic/outbound"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 极速出库
func FastDepartureHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FastOutboundRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := outbound.NewFastDepartureLogic(r.Context(), svcCtx)
		resp, err := l.FastDeparture(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
