package price

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveLogic) Remove(req *types.MaterialPriceRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	_, err = l.svcCtx.MaterialPriceModel.DeleteMany(l.ctx, bson.M{"material": req.Id, "price": req.Price})
	if err != nil {
		fmt.Printf("[Error]删除物料[%s]单价[%.f]:%s\n", req.Id, req.Price, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
