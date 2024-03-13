package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// 出库单
type OutboundOrder struct {
	Id    primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	OldId int64              `json:"old_id" bson:"old_id,omitempty"` //旧系统出库单id
	//出库单状态
	//预发货：出库单尚未确认
	//待拣货：出库单确认后的状态
	//已拣货：出库单物料已拣货
	//待打包：已拣货、已称重维达堡的出库单可以选择打包
	//已打包：出库单已执行打包操作
	//待称重：已拣货、已打包围城中的出库单可以称重
	//已称重：出库单已称重
	//待出库：已拣货、已打包、已称重的出库单可以出库
	//已出库：出库单已出库
	//已签收：出库单已签收
	Status  string `json:"status" bson:"status"`     //出库单状态
	IsPack  int    `json:"is_pack" bson:"is_pack"`   //是否打包：0否,1是
	IsWeigh int    `json:"is_weigh" bson:"is_weigh"` //是否称重：0否,1是
	//出库单类型
	//销售出库
	//样品出库
	//退货出库
	//报废出库
	//赠品出库
	//生产用料出库
	//退料出库
	//损耗出库
	Type          string  `json:"type" bson:"type"`                           //出库单类型
	Code          string  `json:"code" bson:"code"`                           //出库单号
	SupplierId    string  `json:"supplier_id" bson:"supplier_id"`             //供应商id
	SupplierName  string  `json:"supplier_name" bson:"supplier_name"`         //供应商名称
	CustomerId    string  `json:"customer_id" bson:"customer_id"`             //客户id
	CustomerName  string  `json:"customer_name" bson:"customer_name"`         //客户名称
	CarrierId     string  `json:"carrier_id" bson:"carrier_id,omitempty"`     //承运商id
	CarrierName   string  `json:"carrier_name" bson:"carrier_name,omitempty"` //承运商名称
	CarrierCost   float64 `json:"carrier_cost" bson:"carrier_cost"`           //运费
	OtherCost     float64 `json:"other_cost" bson:"other_cost"`               //其他费用
	TotalAmount   float64 `json:"total_amount" bson:"total_amount"`           //物料总金额
	Remark        string  `json:"remark" bson:"remark"`                       //备注
	Annex         string  `json:"annex" bson:"annex"`                         //附件
	Receipt       string  `json:"receipt" bson:"receipt"`                     //签收收据
	CreatorId     string  `json:"creator_id" bson:"creator_id"`               //创建人id
	CreatorName   string  `json:"creator_name" bson:"creator_name"`           //创建人名称
	ConfirmTime   int64   `json:"confirm_time" bson:"confirm_time"`           //出库单确认时间
	PickingTime   int64   `json:"picking_time" bson:"picking_time"`           //拣货时间
	PackingTime   int64   `json:"packing_time" bson:"packing_time"`           //打包时间
	WeighingTime  int64   `json:"weighing_time" bson:"weighing_time"`         //称重时间
	DepartureTime int64   `json:"departure_time" bson:"departure_time"`       //出库时间
	ReceiptTime   int64   `json:"receipt_time" bson:"receipt_time"`           //签收时间
	CreatedAt     int64   `json:"created_at" bson:"created_at"`
	UpdatedAt     int64   `json:"updated_at" bson:"updated_at"`
}
