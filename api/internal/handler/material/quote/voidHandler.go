package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"
)

func VoidHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuoteIdRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewVoidLogic(r.Context(), svcCtx)
		resp, err := l.Void(&req)
		writeJson(w, r, resp, err)
	}
}
