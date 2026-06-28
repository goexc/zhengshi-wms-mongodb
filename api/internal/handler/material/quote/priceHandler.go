package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"
)

func PriceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuotePriceRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewPriceLogic(r.Context(), svcCtx)
		resp, err := l.Price(&req)
		writeJson(w, r, resp, err)
	}
}
