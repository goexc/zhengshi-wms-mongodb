package auth

import (
	"api/pkg/validatorx"
	"errors"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"

	"api/internal/logic/auth"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		//参数校验
		if err := validatorx.Validator.StructCtx(r.Context(), req); err != nil {
			var errs validator.ValidationErrors
			errors.As(err, &errs)
			var es []string
			for _, e := range errs {
				es = append(es, e.Translate(validatorx.Trans))
			}
			var resp = types.BaseResponse{
				Code: http.StatusBadRequest,
				Msg:  strings.Join(es, ", "),
			}

			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}

		l := auth.NewLoginLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
