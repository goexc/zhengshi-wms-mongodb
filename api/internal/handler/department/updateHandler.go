package department

import (
	"api/pkg/validatorx"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"

	"api/internal/logic/department"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DepartmentRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		//参数校验
		if err := validatorx.Validator.StructCtx(r.Context(), req); err != nil {
			errs := err.(validator.ValidationErrors)
			var es []string
			for _, e := range errs {
				es = append(es, e.Translate(validatorx.Trans))
			}
			w.Write([]byte(fmt.Sprintf(`{"code": 400, "msg":"%s"}`, strings.Join(es, "/"))))
			httpx.ErrorCtx(r.Context(), w, nil)
			return
		}

		l := department.NewUpdateLogic(r.Context(), svcCtx)
		resp, err := l.Update(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
