package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// CustomerMaterialDelivery 记录指定客户首次交付指定物料的业务事实。
type CustomerMaterialDelivery struct {
	Id                     primitive.ObjectID `json:"_id" bson:"_id,omitempty"`                                   // 记录id
	CustomerId             string             `json:"customer_id" bson:"customer_id"`                             // 客户id
	CustomerName           string             `json:"customer_name" bson:"customer_name"`                         // 客户名称
	MaterialId             string             `json:"material_id" bson:"material_id"`                             // 物料id
	MaterialName           string             `json:"material_name" bson:"material_name"`                         // 物料名称
	MaterialModel          string             `json:"material_model" bson:"material_model"`                       // 物料型号
	MaterialSpecification  string             `json:"material_specification" bson:"material_specification"`       // 物料规格
	MaterialUnit           string             `json:"material_unit" bson:"material_unit"`                         // 物料单位
	FirstDeliveryTime      int64              `json:"first_delivery_time" bson:"first_delivery_time"`             // 首次交付时间，取出库单 departure_time
	FirstDeliveryOrderCode string             `json:"first_delivery_order_code" bson:"first_delivery_order_code"` // 首次交付出库单号
	FirstDeliveryQuantity  float64            `json:"first_delivery_quantity" bson:"first_delivery_quantity"`     // 首次交付数量
	FirstDeliveryPrice     float64            `json:"first_delivery_price" bson:"first_delivery_price"`           // 首次交付时出库单物料价格
	LastDeliveryTime       int64              `json:"last_delivery_time" bson:"last_delivery_time"`               // 最近一次交付时间
	DeliveryCount          int64              `json:"delivery_count" bson:"delivery_count"`                       // 交付次数
	QuoteStatus            string             `json:"quote_status" bson:"quote_status"`                           // 报价状态：unquoted未报价、quoting报价中、quoted已报价、priced已定价
	LatestQuoteId          string             `json:"latest_quote_id" bson:"latest_quote_id"`                     // 最新报价单id
	LatestQuoteNo          string             `json:"latest_quote_no" bson:"latest_quote_no"`                     // 最新报价单号
	LatestPrice            float64            `json:"latest_price" bson:"latest_price"`                           // 最新报价或最终定价
	SourceValid            bool               `json:"source_valid" bson:"source_valid"`                           // 来源是否仍由当前有效出库单支撑
	SourceInvalidReason    string             `json:"source_invalid_reason" bson:"source_invalid_reason"`         // 来源失效原因
	CreatedAt              int64              `json:"created_at" bson:"created_at"`                               // 创建时间
	UpdatedAt              int64              `json:"updated_at" bson:"updated_at"`                               // 更新时间
}
