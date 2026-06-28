package material

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/mongo"
)

type LatestRebuildDeliveryTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLatestRebuildDeliveryTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LatestRebuildDeliveryTaskLogic {
	return &LatestRebuildDeliveryTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LatestRebuildDeliveryTaskLogic) LatestRebuildDeliveryTask() (resp *types.MaterialDeliveryRebuildTaskResponse, err error) {
	resp = new(types.MaterialDeliveryRebuildTaskResponse)

	task, err := materialdelivery.LatestRebuildTask(l.ctx, l.svcCtx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			resp.Code = http.StatusOK
			resp.Msg = "暂无重建任务"
			return resp, nil
		}
		fmt.Printf("[Error]查询最近客户物料首次交付重建任务:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询最近客户物料首次交付重建任务失败"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = materialDeliveryRebuildTaskToType(task)
	return resp, nil
}
