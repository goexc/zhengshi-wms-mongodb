package handler

import (
	"backend/model"
	"backend/svc"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// 同步客户交易流水
func CustomerTransactionSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()

	//1.查询新系统所有订单
	cur, err := svcCtx.OutboundOrderModel.Find(ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询新系统所有订单：%s\n", err.Error())
		return
	}

	var orders = make([]model.OutboundOrder, 0)
	if err = cur.All(ctx, &orders); err != nil {
		fmt.Printf("[Error]解析新系统订单数据:%s\n", err.Error())
		return
	}

	//2.将订单签收时间、订单金额批量写入客户交易流水表
	//2.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)
	for _, order := range orders {
		filter := bson.D{{"order_code", order.Code}}
		update := bson.D{
			{"$set", bson.D{
				{"type", "应收账款"},
				{"code", ""},
				{"order_code", order.Code},
				{"customer_id", order.CustomerId},
				{"customer_name", order.CustomerName},
				{"amount", order.TotalAmount},
				{"annex", order.Annex},
				{"remark", ""},
				{"time", order.ReceiptTime},
				{"creator", svcCtx.Config.Uid},
				{"creator_name", svcCtx.Config.UserName},
				{"created_at", time.Now().Unix()},
			},
			},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpsert(true)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//2.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	bulkRes, err := svcCtx.CustomerTransactionModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新客户交易流水：%s\n", err.Error())
		return
	}
	fmt.Printf("[Info]批量更新客户交易流水：插入(%d)，更新(%d)，插入(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)
}
