package warehouse_zone

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

func (l *AddLogic) Add(req *types.WarehouseZoneRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//0.仓库是否存在
	warehouseId, err := primitive.ObjectIDFromHex(req.WarehouseId)
	if err != nil {
		fmt.Printf("[Error]仓库[%s]id转换：%s\n", req.WarehouseId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择库区所在仓库"
		return resp, nil
	}

	//激活状态的仓库才可以执行库存管理和操作
	var filter = bson.M{
		"_id":    warehouseId,
		"status": bson.M{"$ne": code.WarehouseStatusCode("删除")},
	}
	warehouseRes := l.svcCtx.WarehouseModel.FindOne(l.ctx, filter)
	switch warehouseRes.Err() {
	case nil:
		var warehouse model.Warehouse
		if err = warehouseRes.Decode(&warehouse); err != nil {
			fmt.Printf("[Error]解析仓库[%s]:%s\n", req.WarehouseId, err.Error())
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
		resp.Msg = "仓库不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询仓库[%s]：%s\n", req.WarehouseId, warehouseRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.同一仓库中，库区名称、库区编号是否占用
	//i 表示不区分大小写
	filter = bson.M{
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
		"warehouse_id": warehouseId,
		"status":       bson.M{"$ne": code.WarehouseZoneStatusCode("删除")},
	}
	singleRes := l.svcCtx.WarehouseZoneModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.WarehouseZone
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复库区:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "库区名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "库区编号已占用"
		default:
			resp.Msg = "库区未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //库区未占用
	default:
		fmt.Printf("[Error]查询重复库区:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.添加库区
	var zone = model.WarehouseZone{
		WarehouseId:  warehouseId,
		Name:         strings.TrimSpace(req.Name),
		Code:         strings.TrimSpace(req.Code),
		Image:        strings.TrimSpace(req.Image),
		Capacity:     req.Capacity,
		CapacityUnit: strings.TrimSpace(req.CapacityUnit),
		Status:       code.WarehouseZoneStatusCode("激活"),
		Manager:      strings.TrimSpace(req.Manager),
		Contact:      strings.TrimSpace(req.Contact),
		Remark:       strings.TrimSpace(req.Remark),
		Creator:      uObjectID,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	_, err = l.svcCtx.WarehouseZoneModel.InsertOne(l.ctx, &zone)
	if err != nil {
		fmt.Printf("[Error]新增库区[%s]:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
