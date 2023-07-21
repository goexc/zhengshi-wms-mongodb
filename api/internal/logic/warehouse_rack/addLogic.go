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
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.WarehouseRackRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//1.库区是否存在
	warehouseZoneId, err := primitive.ObjectIDFromHex(req.WarehouseZoneId)
	if err != nil {
		fmt.Printf("[Error]库区[%s]id转换：%s\n", req.WarehouseZoneId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择货架所在库区"
		return resp, nil
	}

	//激活状态的库区才可以执行库存管理和操作
	var filter = bson.M{
		"_id":    warehouseZoneId,
		"status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")},
	}
	zoneRes := l.svcCtx.WarehouseZoneModel.FindOne(l.ctx, filter)
	var zone model.WarehouseZone
	switch zoneRes.Err() {
	case nil:
		if err = zoneRes.Decode(&zone); err != nil {
			fmt.Printf("[Error]解析库区[%s]:%s\n", req.WarehouseZoneId, err.Error())
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
		fmt.Printf("[Error]查询库区[%s]：%s\n", req.WarehouseZoneId, zoneRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.仓库是否存在
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

	//3.同一库区中，货架名称、货架编号是否占用
	//i 表示不区分大小写
	filter = bson.M{
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
		"warehouse_zone_id": zone.Id,
		"status":            bson.M{"$ne": code.WarehouseRackStatusCode("删除")},
	}
	singleRes := l.svcCtx.WarehouseRackModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.WarehouseRack
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复货架:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "货架名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "货架编号已占用"
		default:
			resp.Msg = "货架未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //货架未占用
	default:
		fmt.Printf("[Error]查询重复货架:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.添加货架
	var rack = model.WarehouseRack{
		WarehouseId:     zone.WarehouseId,
		WarehouseZoneId: zone.Id,
		Type:            code.WarehouseRackTypeCode(strings.TrimSpace(req.Type)),
		Name:            strings.TrimSpace(req.Name),
		Code:            strings.TrimSpace(req.Code),
		Image:           strings.TrimSpace(req.Image),
		Capacity:        req.Capacity,
		CapacityUnit:    strings.TrimSpace(req.CapacityUnit),
		Status:          code.WarehouseRackStatusCode("激活"),
		Contact:         strings.TrimSpace(req.Contact),
		Manager:         strings.TrimSpace(req.Manager),
		Remark:          strings.TrimSpace(req.Remark),
		Creator:         uObjectID,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}

	_, err = l.svcCtx.WarehouseRackModel.InsertOne(l.ctx, &rack)
	if err != nil {
		fmt.Printf("[Error]新增货架[%s]:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
