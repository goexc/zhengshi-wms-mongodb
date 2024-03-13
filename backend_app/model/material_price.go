package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MaterialPrice struct {
	Id       primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Material string             `json:"material" bson:"material"` //物料Id
	Price    float64            `json:"price" bson:"price"`       //物料价格
	//Type        int                `json:"type" bson:"type"`                          //物料类别：1.原料，2.产品//原料的价格不固定，没有比较记录
	CustomerId   string `json:"customer_id" bson:"customer_id"`            //客户id
	CustomerName string `json:"customer_name" bson:"customer_name"`        //客户名称
	Creator      string `json:"creator" bson:"creator"`                    //创建人id
	CreatorName  string `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt    int64  `json:"created_at" bson:"created_at"`
}
