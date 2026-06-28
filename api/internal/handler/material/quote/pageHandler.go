package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"
)

func PageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuotePageRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewPageLogic(r.Context(), svcCtx)
		resp, err := l.Page(&req)
		writeJson(w, r, resp, err)
	}
}
