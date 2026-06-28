package material

import (
	"net/http"

	"api/internal/logic/material"
	"api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func LatestRebuildDeliveryTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := material.NewLatestRebuildDeliveryTaskLogic(r.Context(), svcCtx)
		resp, err := l.LatestRebuildDeliveryTask()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
