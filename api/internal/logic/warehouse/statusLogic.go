package warehouse

import (
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatusLogic {
	return &StatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StatusLogic) Status(req *types.WarehouseStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.仓库是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]仓库[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "仓库参数错误"
		return resp, nil
	}
	//排除已删除的仓库
	filter := bson.M{"_id": id, "status": bson.M{"$ne": 100}}
	count, err := l.svcCtx.WarehouseModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询仓库[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "仓库不存在"
		return resp, nil
	}

	//2.仓库状态修改为“删除”时，应该先检测是否存在下级库区、货架、货位。
	if strings.TrimSpace(req.Status) == "删除" {
		count, err = l.svcCtx.WarehouseZoneModel.CountDocuments(l.ctx, bson.M{
			"warehouse_id": id,
			"status":       bson.M{"$ne": code.WarehouseZoneStatusCode("删除")},
		})
		if err != nil {
			fmt.Printf("[Error]查询仓库[%s]是否存在下级库区:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		if count > 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "请先删除绑定的库区"
			return resp, nil
		}
		count, err = l.svcCtx.WarehouseRackModel.CountDocuments(l.ctx, bson.M{
			"warehouse_id": id,
			"status":       bson.M{"$ne": code.WarehouseRackStatusCode("删除")},
		})
		if err != nil {
			fmt.Printf("[Error]查询仓库[%s]是否存在下级货架:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		if count > 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "请先删除绑定的货架"
			return resp, nil
		}
		count, err = l.svcCtx.WarehouseBinModel.CountDocuments(l.ctx, bson.M{
			"warehouse_id": id,
			"status":       bson.M{"$ne": code.WarehouseBinStatusCode("删除")},
		})
		if err != nil {
			fmt.Printf("[Error]查询仓库[%s]是否存在下级货位:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		if count > 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "请先删除绑定的货位"
			return resp, nil
		}
	}

	//3.更改仓库状态
	var update = bson.M{
		"$set": bson.M{
			"status": code.WarehouseStatusCode(req.Status),
		},
	}
	_, err = l.svcCtx.WarehouseModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改仓库[%s]状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
