package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 入库单
type InboundReceipt struct {
	Id primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	//入库单状态
	//10.待审核：入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
	//20.审核不通过：入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
	//30.审核通过：入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
	//40.未发货：
	//50.在途：
	//60.部分入库：当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
	//70.作废：当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
	//80.入库完成：当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
	Status string `json:"status" bson:"status"` //入库单状态
	//入库单类型
	//采购入库
	//外协入库
	//退货入库
	Type          string            `json:"type" bson:"type"`                     //入库单类型
	Code          string            `json:"code" bson:"code"`                     //入库单号
	SupplierId    string            `json:"supplier_id" bson:"supplier_id"`       //供应商id
	SupplierName  string            `json:"supplier_name" bson:"supplier_name"`   //供应商名称
	CustomerId    string            `json:"customer_id" bson:"customer_id"`       //客户id
	CustomerName  string            `json:"customer_name" bson:"customer_name"`   //客户名称
	TotalAmount   float64           `json:"total_amount" bson:"total_amount"`     //总金额
	ReceivingDate int64             `json:"receiving_date" bson:"receiving_date"` //入库日期
	Materials     []InboundMaterial `json:"materials" bson:"materials,omitempty"` //物料清单
	Remark        string            `json:"remark" bson:"remark"`                 //备注
	Annex         []string          `json:"annex" bson:"annex"`                   //附件
	CreatorId     string            `json:"creator_id" bson:"creator_id"`         //创建人id
	CreatorName   string            `json:"creator_name" bson:"creator_name"`     //创建人名称
	EditorId      string            `json:"editor_id" bson:"editor_id"`           //修改人id
	EditorName    string            `json:"editor_name" bson:"editor_name"`       //修改人名称
	CreatedAt     int64             `json:"created_at" bson:"created_at"`
	UpdatedAt     int64             `json:"updated_at" bson:"updated_at"`
}

type InboundMaterial struct {
	Id                string  `json:"id" bson:"id"`                                 //物料id
	Index             int     `json:"index" bson:"index"`                           //物料顺序
	Price             float64 `json:"price" bson:"price"`                           //物料单价
	Name              string  `json:"name" bson:"name"`                             //物料名称：包括型号、材质、规格、表面处理、强度等级等
	Model             string  `json:"model" bson:"model"`                           //型号：用于唯一标识和区分不同种类的钢材。
	EstimatedQuantity float64 `json:"estimated_quantity" bson:"estimated_quantity"` //预计入库数量
	ActualQuantity    float64 `json:"actual_quantity" bson:"actual_quantity"`       //实际入库数量
	Unit              string  `json:"unit" bson:"unit"`                             //计量单位
	//物料状态：
	//40.未发货
	//50.在途
	//60.部分入库
	//70.作废
	//80.入库完成
	//Status int `json:"status" bson:"status"` //物料状态
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
