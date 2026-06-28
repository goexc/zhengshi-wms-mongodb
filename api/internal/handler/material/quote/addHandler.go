package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"
)

func AddHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuoteSaveRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewAddLogic(r.Context(), svcCtx)
		resp, err := l.Add(&req)
		writeJson(w, r, resp, err)
	}
}
