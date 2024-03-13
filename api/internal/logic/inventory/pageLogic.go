package inventory

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.InventorysRequest) (resp *types.InventorysResponse, err error) {
	resp = new(types.InventorysResponse)

	//1.筛选条件
	var filter = bson.M{}

	//1.0 可用数量
	filter["available_quantity"] = bson.M{"$gt": 0}

	//1.1 入库单类型
	if req.Type != "" {
		filter["type"] = req.Type
	}

	//1.2 物料名称
	if req.MaterialName != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + req.MaterialName + ".*", Options: "i"}}
		filter["name"] = regex
	}

	//1.3 物料型号
	if req.MaterialModel != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + req.MaterialModel + ".*", Options: "i"}}
		filter["model"] = regex
	}

	//1.4 仓库id
	if req.WarehouseId != "" {
		filter["warehouse_id"] = req.WarehouseId
	}

	//1.5 库区id
	if req.WarehouseZoneId != "" {
		filter["warehouse_zone_id"] = req.WarehouseZoneId
	}

	//1.6 货架id
	if req.WarehouseRackId != "" {
		filter["warehouse_rack_id"] = req.WarehouseRackId
	}

	//1.7 货位id
	if req.WarehouseBinId != "" {
		filter["warehouse_bin_id"] = req.WarehouseBinId
	}

	var opt = options.Find().SetSort(bson.M{"created_at": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)

	//2.查询数据
	cur, err := l.svcCtx.InventoryModel.Find(l.ctx, filter, opt)
	if err != nil {
		fmt.Printf("[Error]查询库存分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var inventorys = make([]model.Inventory, 0)
	if err = cur.All(l.ctx, &inventorys); err != nil {
		fmt.Printf("[Error]解析库存分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	count, err := l.svcCtx.InventoryModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询库存分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.1 没有任何记录
	if count == 0 {
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	//3.处理数据
	var list = make([]types.InventoryItem, 0)
	for _, one := range inventorys {
		list = append(list, types.InventoryItem{
			Id:                one.Id.Hex(),
			Type:              one.Type,
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

	//4.查询符合条件的物料的库存总数
	var matchStage = bson.D{{"$match", filter}}
	var groupStage = bson.D{
		{"$group", bson.D{
			{"_id", nil},
			//{"total", bson.D{{"$sum", "$quantity"}}},
			{"total", bson.D{{"$sum", "$available_quantity"}}},
		}},
	}

	var pipeline = mongo.Pipeline{
		matchStage,
		groupStage,
	}

	cur, err = l.svcCtx.InventoryModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询符合条件的物料的库存总数:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	// 解码结果
	var result []bson.M
	if err = cur.All(l.ctx, &result); err != nil {
		log.Fatal(err)
	}
	// 输出库存总数量
	var quantity float64
	if len(result) > 0 {
		quantity = result[0]["total"].(float64)
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = types.InventoryPaginate{
		Total:    count,
		Quantity: quantity,
		List:     list,
	}

	return resp, nil
}
