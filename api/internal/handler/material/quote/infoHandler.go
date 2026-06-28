package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"
)

func InfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuoteIdRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewInfoLogic(r.Context(), svcCtx)
		resp, err := l.Info(&req)
		writeJson(w, r, resp, err)
	}
}
