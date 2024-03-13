package handler

import (
	"backend/model_mysql"
	"backend/svc"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// 同步客户数据
func CustomerSync(svcCtx *svc.ServiceContext) {
	var ctx = context.Background()

	//1.查询旧系统所有客户
	customers, err := svcCtx.OldCustomerModel.FindByPage(ctx, "")
	if err != nil {
		if errors.Is(err, model_mysql.ErrNotFound) {
			fmt.Printf("[Error]旧系统没有客户数据\n")
			return
		}

		fmt.Printf("[Error]查询旧系统所有客户：%s\n", err.Error())
		return
	}

	for idx, one := range customers {
		fmt.Printf("[Info][%d]客户[%s]\n", idx+1, one.Name)
	}

	uid, _ := primitive.ObjectIDFromHex(svcCtx.Config.Uid)

	//2.更新到新系统
	//2.1 构建批量更新的过滤条件
	var bulkWrites []mongo.WriteModel
	for _, one := range customers {
		filter := bson.D{{"name", one.Name}}
		update := bson.D{
			{"$set", bson.D{
				{"name", one.Name},
				{"type", "企业"},
				{"code", ""},
				{"image", ""},
				{"legal_representative", ""},
				{"unified_social_credit_identifier", ""},
				{"address", ""},
				{"contact", ""},
				{"manager", ""},
				{"email", ""},
				{"level", 1},
				{"remark", ""},
				{"status", "活动"},
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
	bulkRes, err := svcCtx.CustomerModel.BulkWrite(ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新客户数据：%s\n", err.Error())
		return
	}

	fmt.Printf("[Info]批量更新客户数据：插入(%d)，更新(%d)，总计(%d)\n", bulkRes.InsertedCount, bulkRes.ModifiedCount, bulkRes.UpsertedCount)
	return
}
