package personal

import (
	"net/http"

	"api/internal/logic/personal"
	"api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func MenuHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := personal.NewMenuLogic(r.Context(), svcCtx)
		resp, err := l.Menu()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
