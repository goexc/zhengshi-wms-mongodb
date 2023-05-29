package {{.PkgName}}

import (
    {{if .HasRequest}}
	"api/pkg/validatorx"
	"fmt"
    "github.com/go-playground/validator/v10"
    "strings"
    {{end}}"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	{{.ImportPackages}}
)

func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{if .HasRequest}}var req types.{{.RequestType}}
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

		{{end}}l := {{.LogicName}}.New{{.LogicType}}(r.Context(), svcCtx)
		{{if .HasResp}}resp, {{end}}err := l.{{.Call}}({{if .HasRequest}}&req{{end}})
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			{{if .HasResp}}httpx.OkJsonCtx(r.Context(), w, resp){{else}}httpx.Ok(w){{end}}
		}
	}
}
