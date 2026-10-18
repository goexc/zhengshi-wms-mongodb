// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package acceptance

import (
	"net/http"

	"api/internal/logic/customer/acceptance"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 客户承兑分页
func PageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CustomerDocumentPageRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := acceptance.NewPageLogic(r.Context(), svcCtx)
		resp, err := l.Page(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
