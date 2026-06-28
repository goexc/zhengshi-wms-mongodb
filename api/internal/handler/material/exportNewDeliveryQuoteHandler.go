// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package material

import (
	"net/http"
	"net/url"
	"strings"

	"api/internal/logic/material"
	"api/internal/svc"
	"api/internal/types"
	"api/pkg/validatorx"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 导出客户新增物料报价
func ExportNewDeliveryQuoteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.NewCustomerMaterialExportRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := validatorx.Validator.StructCtx(r.Context(), req); err != nil {
			errs := err.(validator.ValidationErrors)
			var es []string
			for _, e := range errs {
				es = append(es, e.Translate(validatorx.Trans))
			}
			httpx.OkJsonCtx(r.Context(), w, types.BaseResponse{
				Code: http.StatusBadRequest,
				Msg:  strings.Join(es, ", "),
			})
			return
		}

		l := material.NewExportNewDeliveryQuoteLogic(r.Context(), svcCtx)
		fileName, body, resp, err := l.ExportNewDeliveryQuote(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if resp.Code != http.StatusOK {
			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="new-material-quotes.csv"; filename*=UTF-8''`+url.PathEscape(fileName))
		_, _ = w.Write(body)
	}
}
