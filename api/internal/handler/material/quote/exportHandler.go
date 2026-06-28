package quote

import (
	"net/http"

	quotelogic "api/internal/logic/material/quote"
	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ExportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MaterialQuoteIdRequest
		if !parseAndValidate(w, r, &req) {
			return
		}

		l := quotelogic.NewExportLogic(r.Context(), svcCtx)
		fileName, body, resp, err := l.Export(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if resp.Code != http.StatusOK {
			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
		_, _ = w.Write(body)
	}
}
