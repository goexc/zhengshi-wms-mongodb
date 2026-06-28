package material

import (
	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type RebuildDeliveryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRebuildDeliveryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RebuildDeliveryLogic {
	return &RebuildDeliveryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RebuildDeliveryLogic) RebuildDelivery() (resp *types.MaterialDeliveryRebuildTaskResponse, err error) {
	resp = new(types.MaterialDeliveryRebuildTaskResponse)

	task, created, err := materialdelivery.EnqueueRebuildTask(l.ctx, l.svcCtx, contextString(l.ctx, "uid"), contextString(l.ctx, "name"))
	if err != nil {
		fmt.Printf("[Error]创建客户物料首次交付重建任务:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "创建客户物料首次交付重建任务失败"
		return resp, nil
	}

	resp.Code = http.StatusOK
	if created {
		resp.Msg = "重建任务已提交，系统将在后台执行"
	} else {
		resp.Msg = "已有重建任务正在执行或等待执行"
	}
	resp.Data = materialDeliveryRebuildTaskToType(task)
	return resp, nil
}

func materialDeliveryRebuildTaskToType(task model.MaterialDeliveryRebuildTask) types.MaterialDeliveryRebuildTask {
	id := ""
	if !task.Id.IsZero() {
		id = task.Id.Hex()
	}
	return types.MaterialDeliveryRebuildTask{
		Id:            id,
		Status:        task.Status,
		OrderCount:    task.OrderCount,
		DeliveryCount: task.DeliveryCount,
		Message:       task.Message,
		ErrorMessage:  task.ErrorMessage,
		CreatorId:     task.CreatorId,
		CreatorName:   task.CreatorName,
		CreatedAt:     task.CreatedAt,
		StartedAt:     task.StartedAt,
		FinishedAt:    task.FinishedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}

func contextString(ctx context.Context, key string) string {
	value, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return value
}
