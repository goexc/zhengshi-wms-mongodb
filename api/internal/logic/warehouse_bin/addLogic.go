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

func (l *AddLogic) Add(req *types.WarehouseBinRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}
	//1.所属货架查询[是否存在、状态是否激活]
	warehouseRackId, err := primitive.ObjectIDFromHex(req.WarehouseRackId)
	if err != nil {
		fmt.Printf("[Error]货架[%s]id转换：%s\n", req.WarehouseRackId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择货位所在货架"
		return resp, nil
	}

	//激活状态的货架才可以执行库存管理和操作
	var filter = bson.M{
		"_id":    warehouseRackId,
		"status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")},
	}
	rackRes := l.svcCtx.WarehouseRackModel.FindOne(l.ctx, filter)
	var rack model.WarehouseRack
	switch rackRes.Err() {
	case nil:
		if err = rackRes.Decode(&rack); err != nil {
			fmt.Printf("[Error]解析货架[%s]:%s\n", req.WarehouseRackId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		//货架是否在激活状态
		switch rack.Status {
		case 10: //激活
		default: //非激活状态不能执行库存管理和操作
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("货架%s，无法执行操作", code.WarehouseRackStatusText(rack.Status))
			return resp, nil
		}
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "所属货架不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询货架[%s]：%s\n", req.WarehouseRackId, rackRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.根据货架信息，查询库区、仓库信息[是否存在、状态是否激活]
	//2.1 激活状态的库区才可以执行库存管理和操作
	filter = bson.M{
		"_id":    rack.WarehouseZoneId,
		"status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")},
	}
	zoneRes := l.svcCtx.WarehouseZoneModel.FindOne(l.ctx, filter)
	var zone model.WarehouseZone
	switch zoneRes.Err() {
	case nil:
		if err = zoneRes.Decode(&zone); err != nil {
			fmt.Printf("[Error]解析库区[%s]:%s\n", rack.WarehouseZoneId.Hex(), err.Error())
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
	//2.2 仓库是否存在
	//激活状态的仓库才可以执行库存管理和操作
	filter = bson.M{
		"_id":    rack.WarehouseId,
		"status": bson.M{"$ne": code.WarehouseStatusCode("删除")},
	}
	warehouseRes := l.svcCtx.WarehouseModel.FindOne(l.ctx, filter)
	switch warehouseRes.Err() {
	case nil:
		var warehouse model.Warehouse
		if err = warehouseRes.Decode(&warehouse); err != nil {
			fmt.Printf("[Error]解析仓库[%s]:%s\n", rack.WarehouseId.Hex(), err.Error())
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
		fmt.Printf("[Error]查询仓库[%s]：%s\n", rack.WarehouseId.Hex(), warehouseRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.查询货位名称、货位编号在所属货架中是否占用
	//i 表示不区分大小写
	filter = bson.M{
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
		"warehouse_rack_id": rack.Id,
		"status":            bson.M{"$ne": code.WarehouseBinStatusCode("删除")},
	}
	singleRes := l.svcCtx.WarehouseBinModel.FindOne(l.ctx, filter)
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
		fmt.Printf("[Error]查询重复货位:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.添加货位
	var bin = model.WarehouseBin{
		Id:              primitive.ObjectID{},
		WarehouseId:     rack.WarehouseId,
		WarehouseZoneId: rack.WarehouseZoneId,
		WarehouseRackId: rack.Id,
		Name:            strings.TrimSpace(req.Name),
		Code:            strings.TrimSpace(req.Code),
		Capacity:        req.Capacity,
		CapacityUnit:    strings.TrimSpace(req.CapacityUnit),
		Status:          code.WarehouseBinStatusCode("激活"),
		Remark:          strings.TrimSpace(req.Remark),
		Creator:         uObjectID,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}

	_, err = l.svcCtx.WarehouseBinModel.InsertOne(l.ctx, &bin)
	if err != nil {
		fmt.Printf("[Error]新增货位[%s]:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
