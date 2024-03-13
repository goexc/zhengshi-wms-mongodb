package warehouse_rack

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func (l *StatusLogic) Status(req *types.WarehouseRackStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.货架是否存在
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		fmt.Printf("[Error]货架[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择货架"
		return resp, nil
	}
	//i 表示不区分大小写
	filter := bson.M{
		"_id":    id,
		"status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")},
	}
	singleRes := l.svcCtx.WarehouseRackModel.FindOne(l.ctx, filter)
	var rack model.WarehouseRack
	switch singleRes.Err() {
	case nil: //货架存在
		if err = singleRes.Decode(&rack); err != nil {
			fmt.Printf("[Error]解析重复货架:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //货架不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "货架不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询货架[%s]:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.库区是否存在
	//激活状态的库区才可以执行库存管理和操作
	filter = bson.M{
		"_id":    rack.WarehouseZoneId,
		"status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")},
	}
	zoneRes := l.svcCtx.WarehouseZoneModel.FindOne(l.ctx, filter)
	var zone model.WarehouseZone
	switch zoneRes.Err() {
	case nil:
		if err = zoneRes.Decode(&zone); err != nil {
			fmt.Printf("[Error]解析库区[%s]:%s\n", rack.WarehouseZoneId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		//库区是否在激活状态
		switch zone.Status {
		case 10: //激活
		default: //非激活状态不能执行库存管理和操作
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("库区%s，无法执行操作", code.WarehouseZoneStatusText(zone.Status))
			return resp, nil
		}
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "所属库区不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询库区[%s]：%s\n", rack.WarehouseZoneId.Hex(), zoneRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.仓库是否存在
	//激活状态的仓库才可以执行库存管理和操作
	filter = bson.M{
		"_id":    zone.WarehouseId,
		"status": bson.M{"$ne": code.WarehouseStatusCode("删除")},
	}
	warehouseRes := l.svcCtx.WarehouseModel.FindOne(l.ctx, filter)
	switch warehouseRes.Err() {
	case nil:
		var warehouse model.Warehouse
		if err = warehouseRes.Decode(&warehouse); err != nil {
			fmt.Printf("[Error]解析仓库[%s]:%s\n", zone.WarehouseId.Hex(), err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		//仓库是否在激活状态
		switch warehouse.Status {
		case 10: //激活
		default: //非激活状态不能执行库存管理和操作
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("仓库%s，无法执行操作", code.WarehouseStatusText(warehouse.Status))
			return resp, nil
		}
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "所属仓库不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询仓库[%s]：%s\n", zone.WarehouseId.Hex(), warehouseRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.货架状态修改为“删除”时，应该先检测是否存在下级货位。
	if strings.TrimSpace(req.Status) == "删除" {
		count, e := l.svcCtx.WarehouseBinModel.CountDocuments(l.ctx, bson.M{
			"warehouse_rack_id": id,
			"status":            bson.M{"$ne": code.WarehouseBinStatusCode("删除")},
		})
		if e != nil {
			fmt.Printf("[Error]查询货架[%s]是否存在下级货位:%s\n", req.Id, e.Error())
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

	//5.修改货架状态
	var update = bson.M{
		"$set": bson.M{
			"status": code.WarehouseRackStatusCode(req.Status),
		},
	}
	_, err = l.svcCtx.WarehouseRackModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改货架[%s]状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
