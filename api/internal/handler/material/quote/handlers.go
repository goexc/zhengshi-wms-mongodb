package quote

import (
	"net/http"
	"strings"

	"api/internal/types"
	"api/pkg/validatorx"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func parseAndValidate(w http.ResponseWriter, r *http.Request, req any) bool {
	if err := httpx.Parse(r, req); err != nil {
		httpx.ErrorCtx(r.Context(), w, err)
		return false
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
		return false
	}
	return true
}

func writeJson(w http.ResponseWriter, r *http.Request, resp any, err error) {
	if err != nil {
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	httpx.OkJsonCtx(r.Context(), w, resp)
}
