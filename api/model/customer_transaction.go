package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CustomerTransaction 是客户应收账款的财务流水。
// 余额应由 confirmed 流水汇总得到，customer.receivable_balance 和
// customer.credit_balance 只作为查询缓存。
type CustomerTransaction struct {
	Id                    primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type                  string             `json:"type" bson:"type"`                                       //交易类型展示文案，兼容历史中文数据，不参与索引
	TransactionType       string             `json:"transaction_type" bson:"transaction_type"`               //交易类型 code：opening_ar、outbound_ar、payment 等
	Direction             string             `json:"direction" bson:"direction"`                             //余额方向 code：receivable_increase 或 receivable_decrease
	Status                string             `json:"status" bson:"status"`                                   //流水状态 code：draft、confirmed、reversed、voided
	Code                  string             `json:"code" bson:"code"`                                       //交易编号
	OrderCode             string             `json:"order_code" bson:"order_code"`                           //兼容字段：来源单据编号
	SourceType            string             `json:"source_type" bson:"source_type"`                         //来源类型 code：opening、outbound_order、inbound_return 等
	SourceId              string             `json:"source_id" bson:"source_id"`                             //来源单据 id
	SourceCode            string             `json:"source_code" bson:"source_code"`                         //来源单据编号
	IdempotencyKey        string             `json:"idempotency_key" bson:"idempotency_key"`                 //幂等键，唯一索引使用 ASCII 内容
	OriginalTransactionId string             `json:"original_transaction_id" bson:"original_transaction_id"` //调整或冲销时关联的原流水
	CustomerId            string             `json:"customer_id" bson:"customer_id"`                         //客户id
	CustomerName          string             `json:"customer_name" bson:"customer_name"`                     //客户名称
	Amount                float64            `json:"amount" bson:"amount"`                                   //交易金额
	Annex                 string             `json:"annex" bson:"annex"`                                     //附件
	Remark                string             `json:"remark" bson:"remark"`                                   //备注
	Time                  int64              `json:"time" bson:"time"`                                       //交易时间
	Creator               string             `json:"creator" bson:"creator"`                                 //创建人
	CreatorName           string             `json:"creator_name" bson:"creator_name"`                       //创建人
	Editor                string             `json:"editor" bson:"editor"`                                   //修改人
	EditorName            string             `json:"editor_name" bson:"editor_name"`                         //修改人
	CreatedAt             int64              `json:"created_at" bson:"created_at"`
	UpdatedAt             int64              `json:"updated_at" bson:"updated_at"`
}
