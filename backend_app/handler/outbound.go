package handler

import (
	"backend/model"
	"backend/model_mysql"
	"backend/svc"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

// 出库单数据同步
func OutboundSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()
	uid, _ := primitive.ObjectIDFromHex(svcCtx.Config.Uid)

	//1.查询旧系统所有出库单
	orders, err := svcCtx.OldStockOutModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			fmt.Printf("[Error]旧系统没有出库单数据\n")
			return
		}

		fmt.Printf("[Error]查询旧系统所有出库单：%s\n", err.Error())
		return
	}

	//2.收集出库单客户信息
	var customersMap = make(map[string]bool)
	var customersName = make([]string, 0)
	for _, one := range orders {
		fmt.Printf("[Info]出库单[%s]\n", one.Numbering)
		if _, ok := customersMap[one.ClientName]; !ok {
			customersName = append(customersName, one.ClientName)
			customersMap[one.ClientName] = true
		}
	}

	var customers = make([]model.Customer, 0)
	cur, err := svcCtx.CustomerModel.Find(ctx, bson.M{"name": bson.M{"$in": customersName}})
	if err != nil {
		fmt.Printf("[Error]查询出库单客户信息:%s\n", err.Error())
		return
	}

	defer cur.Close(ctx)

	if err = cur.All(ctx, &customers); err != nil {
		fmt.Printf("[Error]解析出库单客户信息:%s\n", err.Error())
		return
	}

	var customerMap = make(map[string]model.Customer)
	for _, one := range customers {
		customerMap[one.Name] = one
	}

	if len(customerMap) != len(customersName) {
		fmt.Printf("[Error]部分出库单客户信息不存在\n")

		for _, one := range customersName {
			if _, ok := customerMap[one]; !ok {
				fmt.Printf("[Error]出库单客户[%s]不存在\n", one)
			}
		}

		return
	}

	//2.更新到新系统
	//2.1 构建批量更新的过滤条件
	var bulkWrites []mongo.WriteModel
	for _, one := range orders {
		filter := bson.D{{"old_id", one.Id}}
		update := bson.D{
			{"$set", bson.D{
				{"old_id", one.Id},
				{"code", one.Numbering},
				{"status", "已签收"},
				{"is_pack", 0},
				{"is_weigh", 0},
				{"type", "销售出库"},
				{"supplier_id", ""},
				{"supplier_name", ""},
				{"customer_id", customerMap[one.ClientName].Id.Hex()},
				{"customer_name", one.ClientName},
				{"carrier_id", ""},
				{"carrier_name", ""},
				{"carrier_cost", 0},
				{"other_cost", 0},
				{"tax", one.Tax},
				{"total_amount", one.Total},
				{"remark", ""},
				{"annex", ""},
				{"receipt", ""},
				{"creator_id", uid},
				{"creator_name", svcCtx.Config.UserName},
				{"confirm_time", 0},
				{"picking_time", 0},
				{"packing_time", 0},
				{"weighing_time", 0},
				{"departure_time", 0},
				{"receipt_time", one.Date},
				{"created_at", one.CreatedAt.Unix()},
				{"updated_at", time.Now().Unix()},
			}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpdate(update)
		bulkWrite.SetUpsert(true)

		bulkWrites = append(bulkWrites, bulkWrite)

	}
	//2.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	bulkRes, err := svcCtx.OutboundOrderModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新出库单：%s\n", err.Error())
		return
	}
	fmt.Printf("[Info]批量更新出库单：插入(%d)，更新(%d)，总计(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)

}

// 出库单物料数据同步
func OutboundMaterialSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()
	uid, _ := primitive.ObjectIDFromHex(svcCtx.Config.Uid)

	//1.查询旧系统所有出库单物料
	detail, err := svcCtx.OldStockOutDetailModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			fmt.Printf("[Error]旧系统没有出库单物料数据\n")
			return
		}

		fmt.Printf("[Error]查询旧系统所有出库单物料：%s\n", err.Error())
		return
	}

	//2.收集所有物料型号
	var modelMap = make(map[string]bool)
	var models = make([]string, 0)
	for _, one := range detail {
		m := strings.TrimSuffix(strings.TrimPrefix(one.ProductModel, "_"), "_")

		if _, ok := modelMap[m]; !ok {
			modelMap[m] = true
			models = append(models, strings.ReplaceAll(m, "*", "×"))
		}

		if one.ProductModel == "4022" {
			fmt.Println("==物料名称：", one.ProductName, one.ProductModel, one.ProductSpecs)
		}
	}

	//3.查询物料id
	var materials = make([]model.Material, 0)
	cur, err := svcCtx.MaterialModel.Find(ctx, bson.M{"model": bson.M{"$in": models}})
	if err != nil {
		fmt.Printf("[Error]查询出库单物料信息:%s\n", err.Error())
		return
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单物料信息:%s\n", err.Error())
		return
	}

	var materialMap = make(map[string]model.Material)
	for _, one := range materials {
		materialMap[one.Model] = one
		//fmt.Println("物料id：", one.Id.Hex())
	}

	if len(materialMap) != len(models) {
		fmt.Printf("物料数量不一致：%d, %d\n", len(materialMap), len(models))
		fmt.Printf("[Error]部分出库单物料信息不存在\n")
		for _, one := range models {
			if _, ok := materialMap[one]; !ok {
				fmt.Printf("[Error]出库单物料[%s]不存在\n", one)
			}
		}
		return
	}

	//4.查询系统出库单旧id对应的发货单号
	var orders = make([]model.OutboundOrder, 0)
	cur, err = svcCtx.OutboundOrderModel.Find(ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询出库单信息:%s\n", err.Error())
		return
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &orders); err != nil {
		fmt.Printf("[Error]解析出库单信息:%s\n", err.Error())
		return
	}

	var orderCodes = make(map[int64]string)
	for _, one := range orders {
		orderCodes[one.OldId] = one.Code
	}

	//5.判断出库单旧id是否存在
	for _, one := range detail {
		if _, ok := orderCodes[one.StockId]; !ok {
			fmt.Printf("[Error]出库单旧id[%d]不存在\n", one.StockId)
			return
		}
	}

	//2.更新到新系统
	//2.1 构建批量更新的过滤条件
	var bulkWrites []mongo.WriteModel
	for _, one := range detail {
		m := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(one.ProductModel, "_"), "_"), "*", "×")
		fmt.Printf("[Info]物料型号[%s]%s,%s,%s\n", m, materialMap[m].Id.Hex(), materialMap[m].Model, materialMap[m].Specification)
		//if m == "" {
		//	fmt.Println("出库单空物料：", one.ProductName, one.ProductModel, one.ProductSpecs)
		//}

		//filter := bson.D{{"order_code", orderCodes[one.StockId]}, {"material_id", materialMap[m].Id.Hex()}}
		filter := bson.D{{"order_code", orderCodes[one.StockId]}, {"model", materialMap[m].Model}}
		update := bson.D{
			{"$set", bson.D{
				{"old_order_id", one.StockId},
				{"order_code", orderCodes[one.StockId]},
				{"material_id", materialMap[m].Id.Hex()},
				{"index", 0},
				{"name", strings.ReplaceAll(materialMap[m].Name, "*", "×")},
				{"model", strings.ReplaceAll(materialMap[m].Model, "*", "×")},
				{"specification", strings.ReplaceAll(materialMap[m].Specification, "*", "×")},
				{"price", one.Price},
				{"quantity", one.Count},
				{"weight", 0},
				{"unit", materialMap[m].Unit},
				{"inventorys", make([]model.OutboundMaterialInventory, 0)},
				{"creator", uid},
				{"creator_name", svcCtx.Config.UserName},
				{"created_at", one.CreatedAt.Unix()},
				{"updated_at", time.Now().Unix()},
			}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpdate(update)
		bulkWrite.SetUpsert(true)

		bulkWrites = append(bulkWrites, bulkWrite)

	}
	//2.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	bulkRes, err := svcCtx.OutboundMaterialModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新发货单物料数据：%s\n", err.Error())
		return
	}

	fmt.Printf("[Info]批量更新发货单物料数据：插入(%d)，更新(%d)，总计(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)
	return

}
