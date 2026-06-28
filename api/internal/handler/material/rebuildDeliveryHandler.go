package material

import (
	"api/internal/logic/material"
	"api/internal/svc"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func RebuildDeliveryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := material.NewRebuildDeliveryLogic(r.Context(), svcCtx)
		resp, err := l.RebuildDelivery()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
