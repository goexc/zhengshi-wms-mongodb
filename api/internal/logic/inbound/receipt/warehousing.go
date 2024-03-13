package receipt

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// 查询仓储信息
//
//list:map[物料id][]string
func Warehousing(ctx context.Context, svcCtx *svc.ServiceContext, list map[string][]string) (warehouses map[string]model.Warehouse, warehouseZones map[string]model.WarehouseZone, warehouseRacks map[string]model.WarehouseRack, warehouseBins map[string]model.WarehouseBin, err error) {
	//defer func() {
	//	fmt.Println("************仓储信息：", list)
	//	fmt.Println("************仓库地址：", warehouses)
	//	fmt.Println("************库区地址：", warehouseZones)
	//	fmt.Println("************货架地址：", warehouseRacks)
	//	fmt.Println("************货位地址：", warehouseBins)
	//}()

	//物料仓储信息
	var warehousing = make(map[string]types.Warehousing)

	//收集仓库、库区、货架、货位id
	var warehousesId = make([]primitive.ObjectID, 0)
	var warehouseZonesId = make([]primitive.ObjectID, 0)
	var warehouseRacksId = make([]primitive.ObjectID, 0)
	var warehouseBinsId = make([]primitive.ObjectID, 0)

	warehouses = make(map[string]model.Warehouse)         //物料id=>“”
	warehouseZones = make(map[string]model.WarehouseZone) //物料id=>“”
	warehouseRacks = make(map[string]model.WarehouseRack) //物料id=>“”
	warehouseBins = make(map[string]model.WarehouseBin)   //物料id=>“”

	//4.1 收集id
	for materialId, one := range list {
		var warehouseId, zoneId, rackId, binId string
		if len(one) >= 1 {
			warehouseId = one[0]
			warehouseObjectID, _ := primitive.ObjectIDFromHex(one[0]) //warehouseId已通过参数校验，无需再次判断
			warehousesId = append(warehousesId, warehouseObjectID)
		}
		if len(one) >= 2 {
			zoneId = one[1]
			zoneObjectID, _ := primitive.ObjectIDFromHex(one[1]) //warehouseId已通过参数校验，无需再次判断
			warehouseZonesId = append(warehouseZonesId, zoneObjectID)
		}

		if len(one) >= 3 {
			rackId = one[2]
			rackObjectID, _ := primitive.ObjectIDFromHex(one[2]) //warehouseId已通过参数校验，无需再次判断
			warehouseRacksId = append(warehouseRacksId, rackObjectID)
		}

		if len(one) >= 4 {
			binId = one[3]
			binObjectID, _ := primitive.ObjectIDFromHex(one[3]) //warehouseId已通过参数校验，无需再次判断
			warehouseBinsId = append(warehouseBinsId, binObjectID)
		}

		warehousing[materialId] = types.Warehousing{
			WarehouseId:     warehouseId,
			WarehouseZoneId: zoneId,
			WarehouseRackId: rackId,
			WarehouseBinId:  binId,
		}
	}

	var ws = make([]model.Warehouse, 0)
	var zs = make([]model.WarehouseZone, 0)
	var rs = make([]model.WarehouseRack, 0)
	var bs = make([]model.WarehouseBin, 0)

	//4.3.1 查询仓库
	if len(warehousesId) > 0 {
		var cur *mongo.Cursor
		cur, err = svcCtx.WarehouseModel.Find(ctx, bson.M{"_id": bson.M{"$in": warehousesId}, "status": bson.M{"$ne": code.WarehouseStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询仓库列表:%s\n", err.Error())
			return
		}
		defer cur.Close(ctx)

		if err = cur.All(ctx, &ws); err != nil {
			fmt.Printf("[Error]解析仓库列表:%s\n", err.Error())
			return
		}

		for _, one := range ws {
			warehouses[one.Id.Hex()] = one
		}
	}

	//4.3.2 仓库是否存在
	if len(warehousesId) > len(warehouses) {
		err = errors.New("部分仓库不存在")
		return
	}
	for _, id := range warehousesId {
		if one, ok := warehouses[id.Hex()]; ok {
			if one.Status != code.WarehouseStatusCode("激活") {
				err = errors.New(fmt.Sprintf("仓库[%s]%s，请选择其他仓库。", one.Name, code.WarehouseStatusText(one.Status)))
				return
			}
		}
	}

	//4.4.1 查询库区
	if len(warehouseZonesId) > 0 {
		var cur *mongo.Cursor
		cur, err = svcCtx.WarehouseZoneModel.Find(ctx, bson.M{"_id": bson.M{"$in": warehouseZonesId}, "status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询库区列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return
		}
		defer cur.Close(ctx)

		if err = cur.All(ctx, &zs); err != nil {
			fmt.Printf("[Error]解析库区列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return
		}

		for _, one := range zs {
			warehouseZones[one.Id.Hex()] = one
		}
	}
	//4.4.2 库区是否存在
	if len(warehouseZonesId) > len(warehouseZones) {
		err = errors.New("部分库区不存在")
		return
	}

	for _, id := range warehouseZonesId {
		if one, ok := warehouseZones[id.Hex()]; ok {
			if one.Status != code.WarehouseZoneStatusCode("激活") {
				err = errors.New(fmt.Sprintf("库区[%s]%s，请选择其他库区。", one.Name, code.WarehouseZoneStatusText(one.Status)))
				return
			}
		}
	}

	//4.5 查询货架
	if len(warehouseRacksId) > 0 {
		var cur *mongo.Cursor
		cur, err = svcCtx.WarehouseRackModel.Find(ctx, bson.M{"_id": bson.M{"$in": warehouseRacksId}, "status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询货架列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return
		}
		defer cur.Close(ctx)

		if err = cur.All(ctx, &rs); err != nil {
			fmt.Printf("[Error]解析货架列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return
		}

		for _, one := range rs {
			warehouseRacks[one.Id.Hex()] = one
		}
	}
	//4.5.2 货架是否存在
	if len(warehouseRacksId) > len(warehouseRacks) {
		err = errors.New("部分货架不存在")
		return
	}

	for _, id := range warehouseRacksId {
		if one, ok := warehouseRacks[id.Hex()]; ok {
			if one.Status != code.WarehouseRackStatusCode("激活") {
				err = errors.New(fmt.Sprintf("货架[%s]%s，请选择其他货架。", one.Name, code.WarehouseRackStatusText(one.Status)))
				return
			}
		}
	}

	//4.6 查询货位
	if len(warehouseBinsId) > 0 {
		var cur *mongo.Cursor
		cur, err = svcCtx.WarehouseBinModel.Find(ctx, bson.M{"_id": bson.M{"$in": warehouseBinsId}, "status": bson.M{"$ne": code.WarehouseBinStatusCode("删除")}})
		if err != nil {
			fmt.Printf("[Error]查询货位列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return

		}
		defer cur.Close(ctx)

		if err = cur.All(ctx, &bs); err != nil {
			fmt.Printf("[Error]解析货位列表:%s\n", err.Error())
			err = errors.New("服务内部错误")
			return

		}

		for _, one := range bs {
			warehouseBins[one.Id.Hex()] = one
		}
	}
	//4.6.2 货位是否存在
	if len(warehouseBinsId) > len(warehouseBins) {
		err = errors.New("部分货位不存在")
		return
	}

	for _, id := range warehouseBinsId {
		if one, ok := warehouseBins[id.Hex()]; ok {
			if one.Status != code.WarehouseBinStatusCode("激活") {
				err = errors.New(fmt.Sprintf("货位[%s]%s，请选择其他货位。", one.Name, code.WarehouseBinStatusText(one.Status)))
				return
			}
		}
	}

	return
}
