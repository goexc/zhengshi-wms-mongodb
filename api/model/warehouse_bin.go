package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// 货位
type WarehouseBin struct {
	Id              primitive.ObjectID `json:"_id" bson:"_id,omitempty"`                   //
	WarehouseId     primitive.ObjectID `json:"warehouse_id" bson:"warehouse_id"`           //仓库id
	WarehouseZoneId primitive.ObjectID `json:"warehouse_zone_id" bson:"warehouse_zone_id"` //库区id
	WarehouseRackId primitive.ObjectID `json:"warehouse_rack_id" bson:"warehouse_rack_id"` //货架id
	Name            string             `json:"name" bson:"name"`                           // 货位名称
	Code            string             `json:"code" bson:"code"`                           // 货位编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
	Capacity        float64            `json:"capacity" bson:"capacity"`                   // 货位容量
	CapacityUnit    string             `json:"capacity_unit" bson:"capacity_unit"`         // 货位容量单位：面积、体积或其他度量单位
	Remark          string             `json:"remark" bson:"remark"`                       // 备注
	CreatedAt       int64              `json:"created_at" bson:"created_at"`               //
	UpdatedAt       int64              `json:"updated_at" bson:"updated_at"`               //
}
