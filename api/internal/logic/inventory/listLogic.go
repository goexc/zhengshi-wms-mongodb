package inventory

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *types.InventoryListRequest) (resp *types.InventoryListResponse, err error) {
	resp = new(types.InventoryListResponse)

	//1.筛选条件
	var filter = bson.M{"material_id": req.MaterialId, "available_quantity": bson.M{"$gt": 0}}
	var opt = options.Find().SetSort(bson.M{"created_at": -1})

	//2.查询数据
	cur, err := l.svcCtx.InventoryModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询物料库存:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var inventorys = make([]model.Inventory, 0)
	if err = cur.All(l.ctx, &inventorys); err != nil {
		fmt.Printf("[Error]解析物料库存页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.处理数据
	var list = make([]types.InventoryItem, 0)
	for _, one := range inventorys {
		list = append(list, types.InventoryItem{
			Id:                one.Id.Hex(),
			Type:              one.Type,
			EntryTime:         one.EntryTime,
			WarehouseId:       one.WarehouseId,
			WarehouseName:     one.WarehouseName,
			WarehouseZoneId:   one.WarehouseZoneId,
			WarehouseZoneName: one.WarehouseZoneName,
			WarehouseRackId:   one.WarehouseRackId,
			WarehouseRackName: one.WarehouseRackName,
			WarehouseBinId:    one.WarehouseBinId,
			WarehouseBinName:  one.WarehouseBinName,
			ReceiptCode:       one.ReceiptCode,
			ReceiveCode:       one.ReceiveCode,
			MaterialId:        one.MaterialId,
			Name:              one.Name,
			Price:             one.Price,
			Model:             one.Model,
			Unit:              one.Unit,
			Quantity:          one.Quantity,
			AvailableQuantity: one.AvailableQuantity,
			LockedQuantity:    one.LockedQuantity,
			FrozenQuantity:    one.FrozenQuantity,
			CreatorId:         one.CreatorId,
			CreatorName:       one.CreatorName,
			CreatedAt:         one.CreatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = list

	return resp, nil
}
