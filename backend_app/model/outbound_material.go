package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// 出库单物料
type OutboundOrderMaterial struct {
	Id            primitive.ObjectID          `json:"_id" bson:"_id,omitempty"`
	OrderCode     string                      `json:"order_code" bson:"order_code"`       //发货单号
	MaterialId    string                      `json:"material_id" bson:"material_id"`     //物料id
	Index         int                         `json:"index" bson:"index"`                 //物料顺序
	Name          string                      `json:"name" bson:"name"`                   //物料名称：包括型号、材质、规格、表面处理、强度等级等
	Model         string                      `json:"model" bson:"model"`                 //型号：用于唯一标识和区分不同种类的钢材。
	Specification string                      `json:"specification" bson:"specification"` //规格：包括长度、宽度、厚度等尺寸信息。
	Price         float64                     `json:"price" bson:"price"`                 //物料单价
	Quantity      float64                     `json:"quantity" bson:"quantity"`           //出库数量
	Weight        float64                     `json:"weight" bson:"weight"`               //物料称重
	Unit          string                      `json:"unit" bson:"unit"`                   //计量单位
	Inventorys    []OutboundMaterialInventory `json:"inventorys" bson:"inventorys"`       //库存出货数量
}

type OutboundMaterialInventory struct {
	InventoryId      string  `json:"inventory_id" bson:"inventory_id"`           //库存id
	ShipmentQuantity float64 `json:"shipment_quantity" bson:"shipment_quantity"` //出货数量
}
