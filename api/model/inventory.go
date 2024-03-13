package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// 库存
type Inventory struct {
	Id primitive.ObjectID `json:"_id" bson:"_id,omitempty"` //
	//入库单类型
	//采购入库
	//外协入库
	//生产入库
	//退货入库
	Type              string  `json:"type" bson:"type"`                               //入库单类型
	WarehouseId       string  `json:"warehouse_id" bson:"warehouse_id"`               //仓库id
	WarehouseName     string  `json:"warehouse_name" bson:"warehouse_name"`           //仓库名称
	WarehouseZoneId   string  `json:"warehouse_zone_id" bson:"warehouse_zone_id"`     //库区id
	WarehouseZoneName string  `json:"warehouse_zone_name" bson:"warehouse_zone_name"` //库区名称
	WarehouseRackId   string  `json:"warehouse_rack_id" bson:"warehouse_rack_id"`     //货架id
	WarehouseRackName string  `json:"warehouse_rack_name" bson:"warehouse_rack_name"` //货架名称
	WarehouseBinId    string  `json:"warehouse_bin_id" bson:"warehouse_bin_id"`       //货位id
	WarehouseBinName  string  `json:"warehouse_bin_name" bson:"warehouse_bin_name"`   //货位名称
	ReceiptCode       string  `json:"receipt_code" bson:"receipt_code"`               //入库单编号
	ReceiveCode       string  `json:"receive_code" bson:"receive_code"`               //批次入库编号
	EntryTime         int64   `json:"entry_time" bson:"entry_time"`                   //入库时间
	MaterialId        string  `json:"material_id" bson:"material_id"`                 //物料id
	Name              string  `json:"name" bson:"name"`                               //物料名称：包括型号、材质、规格、表面处理、强度等级等
	Price             float64 `json:"price" bson:"price"`                             //物料单价
	Model             string  `json:"model" bson:"model"`                             //型号：用于唯一标识和区分不同种类的钢材。
	Unit              string  `json:"unit" bson:"unit"`                               //计量单位
	Quantity          float64 `json:"quantity" bson:"quantity"`                       //库存数量
	AvailableQuantity float64 `json:"available_quantity" bson:"available_quantity"`   //可用库存数量
	LockedQuantity    float64 `json:"locked_quantity" bson:"locked_quantity"`         //锁定库存数量
	FrozenQuantity    float64 `json:"frozen_quantity" bson:"frozen_quantity"`         //冻结库存数量
	CreatorId         string  `json:"creator_id" bson:"creator_id"`                   //创建人id
	CreatorName       string  `json:"creator_name" bson:"creator_name"`               //创建人名称
	CreatedAt         int64   `json:"created_at" bson:"created_at"`
}
