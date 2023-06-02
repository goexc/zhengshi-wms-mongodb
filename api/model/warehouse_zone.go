package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// 库区
type WarehouseZone struct {
	Id            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`             //
	WarehouseId   primitive.ObjectID `json:"warehouse_id" bson:"warehouse_id"`     //仓库id
	WarehouseName string             `json:"warehouse_name" bson:"warehouse_name"` // 仓库名称
	Name          string             `json:"name" bson:"name"`                     // 库区名称
	Code          string             `json:"code" bson:"code"`                     // 库区编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
	Capacity      float64            `json:"capacity" bson:"capacity"`             // 库区容量
	CapacityUnit  string             `json:"capacity_unit" bson:"capacity_unit"`   // 库区容量单位：面积、体积或其他度量单位
	//库区状态
	//10.激活（Active）：表示库区处于可用状态，可以执行库存管理和操作。
	//20.禁用（Disabled）：表示库区处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用库区。
	//30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的库区。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
	//90.关闭（Closed）：表示货架已经关闭，不再进行任何库存管理和操作。通常是由于货架不再使用或被替代。
	//100.删除（Deleted）
	Status      int                `json:"status" bson:"status"`                      //库区状态
	Manager     string             `json:"manager" bson:"manager"`                    //负责人
	Contact     string             `json:"contact" bson:"contact"`                    //联系方式
	Remark      string             `json:"remark" bson:"remark"`                      // 备注
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at" bson:"created_at"`              //
	UpdatedAt   int64              `json:"updated_at" bson:"updated_at"`              //
}
