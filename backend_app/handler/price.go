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
	"golang.org/x/sync/errgroup"
	"strings"
)

// 同步物料单价数据
func PriceSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()
	uid, _ := primitive.ObjectIDFromHex(svcCtx.Config.Uid)

	//1.查询旧系统所有物料价格
	var priceMap = make(map[int64][]model_mysql.ProductPrice)
	prices, err := svcCtx.ProductPriceModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			return
		}

		fmt.Printf("[Error]查询旧系统所有物料价格：%s\n", err.Error())
		return
	}

	//2.根据旧系统物料id查询物料信息
	var productsId = make(map[int64]bool)
	for _, one := range prices {
		if _, ok := productsId[one.ProductId]; !ok {
			productsId[one.ProductId] = true
		}

		priceMap[one.ProductId] = append(priceMap[one.ProductId], one)
	}

	var products = make([]model_mysql.Product, 0)

	var g errgroup.Group
	g.SetLimit(1)
	for productId := range productsId {
		g.Go(func() error {
			product, e := svcCtx.ProductModel.FindOne(ctx, productId)
			if e != nil {
				if errors.Is(e, model_mysql.ErrNotFound) {
					fmt.Printf("[Error]忽略不存在的产品[%d]:%s\n", productId, e.Error())
					return nil
				}
				fmt.Printf("[Error]查询旧系统物料[%d]:%s\n", productId, e.Error())
				return e
			}

			products = append(products, *product)

			return nil
		})
	}

	if err = g.Wait(); err != nil {
		return
	}

	fmt.Println("可同步价格物料数量：", len(products))
	//var productIdMap = make(map[int64]string) //旧物料id->新物料id
	var productIdMap = make(map[string]string) //旧物料Model->新物料id
	for _, product := range products {
		//if strings.HasPrefix(product.Model, "_") {
		//	fmt.Printf("[Info]忽略物料：%s(%s)\n", product.Model, product.Name)
		//	continue
		//}

		one := product
		var material model.Material
		m := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(product.Model, "_"), "_"), "*", "×")
		//e := svcCtx.MaterialModel.FindOne(ctx, bson.M{"name": one.Name, "model": one.Model}).Decode(&material)
		e := svcCtx.MaterialModel.FindOne(ctx, bson.M{"model": m}).Decode(&material)
		if e != nil {
			if errors.Is(e, mongo.ErrNoDocuments) {
				fmt.Printf("[Error]同步物料价格前，先同步物料:[%d]%s(%s)\n", one.Id, one.Model, one.Name)
				continue
			}

			fmt.Printf("[Error]查询物料[%s(%s)]：%s\n", one.Model, one.Name, e.Error())
			return
		}

		//productIdMap[product.Id] = material.Id.Hex()
		productIdMap[m] = material.Id.Hex()

		//break
	}

	//3.查询旧系统客户列表
	var clients = make([]model_mysql.Client, 0)
	clients, err = svcCtx.OldCustomerModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			return
		}

		fmt.Printf("[Error]查询旧系统所有客户：%s\n", err.Error())
		return
	}

	//3.2 查询客户在新系统的id
	var clientsMap = make(map[int64]model.Customer)
	for _, one := range clients {
		var customer model.Customer
		filter := bson.D{{"name", one.Name}}
		e := svcCtx.CustomerModel.FindOne(ctx, filter).Decode(&customer)
		if e != nil {
			fmt.Println("[Error]新系统查询客户：", one.Name, e.Error())
			return
		}

		clientsMap[one.Id] = customer
	}

	//for id, one := range clientsMap {
	//	fmt.Printf("新系统客户信息[%d][%s][%s]\n", id, one.Id.Hex(), one.Name)
	//}
	//
	//return

	//return
	//4.更新物料价格
	//4.1 构建批量更新的过滤条件
	var bulkWrites []mongo.WriteModel
	for _, product := range products {
		for _, price := range priceMap[product.Id] {
			m := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(product.Model, "_"), "_"), "*", "×")
			filter := bson.D{{"material", productIdMap[m]}, {"price", price.Price}}
			update := bson.D{
				{"$set", bson.D{
					{"material", productIdMap[m]},
					{"price", price.Price},
					{"customer_id", clientsMap[product.ClientId].Id.Hex()},
					{"customer_name", clientsMap[product.ClientId].Name},
					{"creator", uid},
					{"creator_name", svcCtx.Config.UserName},
					{"created_at", product.CreatedAt.Unix()},
				}},
			}

			bulkWrite := mongo.NewUpdateOneModel()
			bulkWrite.SetFilter(filter)
			bulkWrite.SetUpdate(update)
			bulkWrite.SetUpsert(true)

			bulkWrites = append(bulkWrites, bulkWrite)
		}
	}
	//4.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	bulkRes, err := svcCtx.MaterialPriceModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新物料价格数据：%s\n", err.Error())
		return
	}

	fmt.Printf("[Info]批量更新物料价格数据：插入(%d)，更新(%d)，总计(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)

}
