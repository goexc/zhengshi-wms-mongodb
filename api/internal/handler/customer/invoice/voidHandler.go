// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package invoice

import (
	"net/http"

	"api/internal/logic/customer/invoice"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 作废销售开票
func VoidHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CustomerDocumentVoidRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := invoice.NewVoidLogic(r.Context(), svcCtx)
		resp, err := l.Void(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
