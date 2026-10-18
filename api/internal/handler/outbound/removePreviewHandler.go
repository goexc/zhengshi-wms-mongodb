// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package outbound

import (
	"net/http"

	"api/internal/logic/outbound"
	"api/internal/svc"
	"api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 出库单删除预览
func RemovePreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OutboundOrderIdRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := outbound.NewRemovePreviewLogic(r.Context(), svcCtx)
		resp, err := l.RemovePreview(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
