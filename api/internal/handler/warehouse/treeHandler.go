package warehouse

import (
	"net/http"

	"api/internal/logic/warehouse"
	"api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func TreeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := warehouse.NewTreeLogic(r.Context(), svcCtx)
		resp, err := l.Tree()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
