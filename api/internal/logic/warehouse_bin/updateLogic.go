package warehouse_bin

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
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.WarehouseBinRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.货位是否存在
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		fmt.Printf("[Error]货位[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择货位"
		return resp, nil
	}
	//i 表示不区分大小写
	filter := bson.M{
		"_id":    id,
		"status": bson.M{"$ne": code.WarehouseBinStatusCode("删除")},
	}
	singleRes := l.svcCtx.WarehouseBinModel.FindOne(l.ctx, filter)
	var bin model.WarehouseBin
	switch singleRes.Err() {
	case nil: //货位存在
		if err = singleRes.Decode(&bin); err != nil {
			fmt.Printf("[Error]解析重复货位:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //货位不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "货位不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询货位[%s]:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.货架是否存在
	filter = bson.M{
		"_id":    bin.WarehouseRackId,
		"status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")},
	}
	singleRes = l.svcCtx.WarehouseRackModel.FindOne(l.ctx, filter)
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

	//3.库区是否存在
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
			resp.Msg = fmt.Sprintf("所属库区%s，无法执行操作", code.WarehouseZoneStatusText(zone.Status))
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

	//4.仓库是否存在
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
			resp.Msg = fmt.Sprintf("所属仓库%s，无法执行操作", code.WarehouseStatusText(warehouse.Status))
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

	//5.同一货架中，货位名称、货位编号是否占用
	//i 表示不区分大小写
	filter = bson.M{
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
		"warehouse_rack_id": bin.WarehouseRackId,
		"_id":               bson.M{"$ne": id},
		"status":            bson.M{"$ne": code.WarehouseBinStatusCode("删除")},
	}
	singleRes = l.svcCtx.WarehouseBinModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.WarehouseBin
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复货位:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "货位名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "货位编号已占用"
		default:
			resp.Msg = "货位未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //货位未占用
	default:
		fmt.Printf("[Error]查询重复货位[%s][%s]:%s\n", req.Name, req.Code, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.更新货位信息:不更新货位状态
	var update = bson.M{
		"$set": bson.M{
			"name":          strings.TrimSpace(req.Name),
			"code":          strings.TrimSpace(req.Code),
			"capacity":      req.Capacity,
			"capacity_unit": strings.TrimSpace(req.CapacityUnit),
			"remark":        strings.TrimSpace(req.Remark),
			"updated_at":    time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.WarehouseBinModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新货位[%s]信息：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
