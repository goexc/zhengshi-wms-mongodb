package outbound

import (
	"api/pkg/validatorx"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"

	"api/internal/logic/outbound"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SummaryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("接收请求：", r)
		var req types.OutboundSummaryRequest
		if err := httpx.Parse(r, &req); err != nil {
			fmt.Println("请求参数解析失败：", err.Error())
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
			var resp = types.BaseResponse{
				Code: http.StatusBadRequest,
				Msg:  strings.Join(es, ", "),
			}

			fmt.Printf("参数错误：%#v\n", resp)
			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}

		l := outbound.NewSummaryLogic(r.Context(), svcCtx)
		resp, err := l.Summary(&req)
		if err != nil {
			fmt.Println("错误响应：", err)
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			fmt.Println("响应：", resp)
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
