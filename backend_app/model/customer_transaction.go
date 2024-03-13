package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 客户交易流水表
type CustomerTransaction struct {
	Id           primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type         string             `json:"type" bson:"type"`                   //交易类型：应收账款、回款、退货
	Code         string             `json:"code" bson:"code"`                   //交易编号
	OrderCode    string             `json:"order_code" bson:"order_code"`       //订单编号：出库单号（发货出库）、入库单号（退货入库）
	CustomerId   string             `json:"customer_id" bson:"customer_id"`     //客户id
	CustomerName string             `json:"customer_name" bson:"customer_name"` //客户名称
	Amount       float64            `json:"amount" bson:"amount"`               //交易金额
	Annex        string             `json:"annex" bson:"annex"`                 //附件
	Remark       string             `json:"remark" bson:"remark"`               //备注
	Time         int64              `json:"time" bson:"time"`                   //交易时间
	Creator      string             `json:"creator" bson:"creator"`             //创建人
	CreatorName  string             `json:"creator_name" bson:"creator_name"`   //创建人
	Editor       string             `json:"editor" bson:"editor"`               //修改人
	EditorName   string             `json:"editor_name" bson:"editor_name"`     //修改人
	CreatedAt    int64              `json:"created_at" bson:"created_at"`
	UpdatedAt    int64              `json:"updated_at" bson:"updated_at"`
}
