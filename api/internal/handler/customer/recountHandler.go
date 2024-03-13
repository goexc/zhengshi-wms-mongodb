package customer

import (
	"net/http"

	"api/internal/logic/customer"
	"api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RecountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := customer.NewRecountLogic(r.Context(), svcCtx)
		resp, err := l.Recount()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
