package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 入库单
type Inbound struct {
	Id         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Code       string             `json:"code" bson:"code"`                     //入库单号
	Order      string             `json:"order" bson:"order"`                   //订单编号：表示关联的采购订单编号
	SupplierId primitive.ObjectID `json:"supplier_id" bson:"supplier_id"`       //供应商
	Status     string             `json:"status"`                               //入库单状态：待审核、审核不通过、审核通过
	Materials  []InboundMaterial  `json:"materials" bson:"materials,omitempty"` //物料清单
	Remark     string             `json:"remark" bson:"remark"`                 //备注
	CreatedAt  int64              `json:"created_at" bson:"created_at"`
	UpdatedAt  int64              `json:"updated_at" bson:"updated_at"`
}

type InboundMaterial struct {
	Index        int                `json:"index" bson:"index"`                 //物料顺序
	MaterialId   primitive.ObjectID `json:"material_id" bson:"material_id"`     //物料id
	MaterialName string             `json:"material_name" bson:"material_name"` //物料名称
	Quantity     float64            `json:"quantity" bson:"quantity"`           //物料数量
	Unit         string             `json:"unit" bson:"unit"`                   //计量单位
}
