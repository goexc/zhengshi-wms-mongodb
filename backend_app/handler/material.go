package handler

import (
	"backend/model_mysql"
	"backend/svc"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

func MaterialSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()

	//1.查询旧系统所有物料
	products, err := svcCtx.ProductModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			return
		}

		fmt.Printf("[Error]查询旧系统所有物料：%s\n", err.Error())
		return
	}

	//2.物料去重
	var productsMap = make(map[string]model_mysql.Product)
	for _, one := range products {
		m := strings.TrimSuffix(strings.TrimPrefix(one.Model, "_"), "_")
		if _, ok := productsMap[m]; !ok {
			productsMap[m] = one
		} else {
			fmt.Printf("[Error]重复物料：%s(%s)\n", one.Model, one.Name)
		}
	}

	//2.更新到新系统
	//2.1 构建批量更新的过滤条件
	var bulkWrites []mongo.WriteModel
	for _, one := range productsMap {

		filter := bson.D{{"name", one.Name}, {"model", one.Model}}
		update := bson.D{
			{"$set", bson.D{
				{"old_id", one.Id},
				{"name", strings.ReplaceAll(one.Name, "*", "×")},
				{"model", strings.ReplaceAll(one.Model, "*", "×")},
				{"specification", strings.ReplaceAll(one.Specs, "*", "×")},
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
	bulkRes, err := svcCtx.MaterialModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新物料数据：%s\n", err.Error())
		return
	}

	fmt.Printf("[Info]批量更新物料数据：插入(%d)，更新(%d)，总计(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)

}
