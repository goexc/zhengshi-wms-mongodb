package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 批次入库记录
type InboundReceive struct {
	Id               primitive.ObjectID       `json:"_id" bson:"_id,omitempty"`
	InboundReceiptId string                   `json:"inbound_receipt_id" bson:"inbound_receipt_id"` //入库单id
	Code             string                   `json:"code" bson:"code"`                             //批次入库编号
	CarrierId        string                   `json:"carrier_id" bson:"carrier_id"`                 //承运商id
	CarrierName      string                   `json:"carrier_name" bson:"carrier_name"`             //承运商名称
	CarrierCost      float64                  `json:"carrier_cost" bson:"carrier_cost"`             //运费
	OtherCost        float64                  `json:"other_cost" bson:"other_cost"`                 //其他费用
	TotalAmount      float64                  `json:"total_amount" bson:"total_amount"`             //批次入库物料总金额
	ReceivingDate    int64                    `json:"receiving_date" bson:"receiving_date"`         //入库日期
	Materials        []InboundReceiveMaterial `json:"materials" bson:"materials,omitempty"`         //批次入库物料清单
	Annex            []string                 `json:"annex" bson:"annex"`                           //附件
	Remark           string                   `json:"remark" bson:"remark"`                         //备注
	CreatorId        string                   `json:"creator_id" bson:"creator_id"`                 //创建人id
	CreatorName      string                   `json:"creator_name" bson:"creator_name"`             //创建人名称
	CreatedAt        int64                    `json:"created_at" bson:"created_at"`
}

type InboundReceiveMaterial struct {
	Id             string  `json:"id" bson:"id"`                           //物料id
	Index          int     `json:"index" bson:"index"`                     //物料顺序
	Price          float64 `json:"price" bson:"price"`                     //物料单价
	Name           string  `json:"name" bson:"name"`                       //物料名称：包括型号、材质、规格、表面处理、强度等级等
	Model          string  `json:"model" bson:"model"`                     //型号：用于唯一标识和区分不同种类的钢材。
	Unit           string  `json:"unit" bson:"unit"`                       //计量单位
	ActualQuantity float64 `json:"actual_quantity" bson:"actual_quantity"` //实际入库数量
	//物料状态：
	//60.部分入库
	//70.作废
	//80.入库完成
	Status string `json:"status" bson:"status"` //物料状态

	WarehouseId       string `json:"warehouse_id" bson:"warehouse_id"`               //仓库id
	WarehouseName     string `json:"warehouse_name" bson:"warehouse_name"`           //仓库名称
	WarehouseZoneId   string `json:"warehouse_zone_id" bson:"warehouse_zone_id"`     //库区id
	WarehouseZoneName string `json:"warehouse_zone_name" bson:"warehouse_zone_name"` //库区名称
	WarehouseRackId   string `json:"warehouse_rack_id" bson:"warehouse_rack_id"`     //货架id
	WarehouseRackName string `json:"warehouse_rack_name" bson:"warehouse_rack_name"` //货架名称
	WarehouseBinId    string `json:"warehouse_bin_id" bson:"warehouse_bin_id"`       //货位id
	WarehouseBinName  string `json:"warehouse_bin_name" bson:"warehouse_bin_name"`   //货位名称
}
