package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Plan struct {
	Id               primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type             string             `json:"type" bson:"type"`                           //类型:采购计划、生产计划
	Status           string             `json:"status" bson:"status"`                       //状态:执行中、已完成
	CustomerId       string             `json:"customer_id" bson:"customer_id"`             //客户id
	CustomerName     string             `json:"customer_name" bson:"customer_name"`         //客户名称
	SupplierId       string             `json:"supplier_id" bson:"supplier_id"`             //供应商id
	SupplierName     string             `json:"supplier_name" bson:"supplier_name"`         //供应商名称
	MaterialId       string             `json:"material_id" bson:"material_id"`             //物料id
	MaterialName     string             `json:"material_name" bson:"material_name"`         //物料名称：包括型号、材质、规格、表面处理、强度等级等
	MaterialImage    string             `json:"material_image" bson:"material_image"`       //物料图片
	MaterialModel    string             `json:"material_model" bson:"material_model"`       //型号：用于唯一标识和区分不同种类的钢材。
	MaterialUnit     string             `json:"material_unit" bson:"material_unit"`         //计量单位
	MaterialQuantity float64            `json:"material_quantity" bson:"material_quantity"` //数量
	Deadline         int64              `json:"deadline" bson:"deadline"`                   //截止日期
	CreatorId        string             `json:"creator_id" bson:"creator_id"`               //创建人id
	CreatorName      string             `json:"creator_name" bson:"creator_name"`           //创建人名称
	CreatedAt        int64              `json:"created_at" bson:"created_at,omitempty"`
}
