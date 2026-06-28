package material

import (
	"context"
	"fmt"
	"net/http"

	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"
	quoteCode "api/pkg/code"

	"github.com/zeromicro/go-zero/core/logx"
)

type RebuildDeliveryTasksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRebuildDeliveryTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RebuildDeliveryTasksLogic {
	return &RebuildDeliveryTasksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RebuildDeliveryTasksLogic) RebuildDeliveryTasks(req *types.MaterialDeliveryRebuildTaskRequest) (resp *types.MaterialDeliveryRebuildTaskPageResponse, err error) {
	resp = new(types.MaterialDeliveryRebuildTaskPageResponse)

	if req.Status != "" && !quoteCode.ValidMaterialDeliveryRebuildTaskStatus(req.Status) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "任务状态不正确"
		return resp, nil
	}

	tasks, total, err := materialdelivery.ListRebuildTasks(l.ctx, l.svcCtx, req.Status, req.Page, req.Size)
	if err != nil {
		fmt.Printf("[Error]查询客户物料首次交付重建任务:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询客户物料首次交付重建任务失败"
		return resp, nil
	}

	list := make([]types.MaterialDeliveryRebuildTask, 0, len(tasks))
	for _, task := range tasks {
		list = append(list, materialDeliveryRebuildTaskToType(task))
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Total = total
	resp.Data.List = list
	return resp, nil
}
