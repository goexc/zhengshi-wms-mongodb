package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// MaterialQuote 保存客户新增物料的报价过程，最终定价仍沉淀到 material_price。
type MaterialQuote struct {
	Id                    primitive.ObjectID      `json:"_id" bson:"_id,omitempty"`                             // 报价单id
	QuoteNo               string                  `json:"quote_no" bson:"quote_no"`                             // 报价单号
	CustomerId            string                  `json:"customer_id" bson:"customer_id"`                       // 客户id
	CustomerName          string                  `json:"customer_name" bson:"customer_name"`                   // 客户名称
	MaterialId            string                  `json:"material_id" bson:"material_id"`                       // 物料id
	MaterialName          string                  `json:"material_name" bson:"material_name"`                   // 物料名称
	MaterialModel         string                  `json:"material_model" bson:"material_model"`                 // 物料型号
	MaterialSpecification string                  `json:"material_specification" bson:"material_specification"` // 物料规格
	MaterialUnit          string                  `json:"material_unit" bson:"material_unit"`                   // 物料单位
	DeliveryId            string                  `json:"delivery_id" bson:"delivery_id"`                       // 客户物料首次交付记录id
	SourceOrderCode       string                  `json:"source_order_code" bson:"source_order_code"`           // 来源首次交付出库单号
	QuoteMode             string                  `json:"quote_mode" bson:"quote_mode"`                         // 报价方式：detailed详细报价、simple简单报价
	Status                string                  `json:"status" bson:"status"`                                 // 报价状态：draft草稿、submitted已提交、quoted已报价、priced已定价、void作废
	Currency              string                  `json:"currency" bson:"currency"`                             // 币种，默认 CNY
	CostItems             []MaterialQuoteCostItem `json:"cost_items" bson:"cost_items"`                         // 详细报价成本项
	SimplePrice           float64                 `json:"simple_price" bson:"simple_price"`                     // 简单报价单价
	TotalCost             float64                 `json:"total_cost" bson:"total_cost"`                         // 成本合计
	ProfitRate            float64                 `json:"profit_rate" bson:"profit_rate"`                       // 利润率
	ProfitAmount          float64                 `json:"profit_amount" bson:"profit_amount"`                   // 利润金额
	TaxRate               float64                 `json:"tax_rate" bson:"tax_rate"`                             // 税率
	TaxAmount             float64                 `json:"tax_amount" bson:"tax_amount"`                         // 税额
	FinalPrice            float64                 `json:"final_price" bson:"final_price"`                       // 最终报价单价
	TotalAmount           float64                 `json:"total_amount" bson:"total_amount"`                     // 兼容字段，当前等于最终报价
	ValidFrom             int64                   `json:"valid_from" bson:"valid_from"`                         // 报价有效开始时间
	ValidTo               int64                   `json:"valid_to" bson:"valid_to"`                             // 报价有效结束时间
	Remark                string                  `json:"remark" bson:"remark"`                                 // 备注
	SourceValid           bool                    `json:"source_valid" bson:"source_valid"`                     // 来源首次交付记录是否有效
	SourceInvalidReason   string                  `json:"source_invalid_reason" bson:"source_invalid_reason"`   // 来源失效原因
	CreatorId             string                  `json:"creator_id" bson:"creator_id"`                         // 创建人id
	CreatorName           string                  `json:"creator_name" bson:"creator_name"`                     // 创建人名称
	CreatedAt             int64                   `json:"created_at" bson:"created_at"`                         // 创建时间
	UpdatedAt             int64                   `json:"updated_at" bson:"updated_at"`                         // 更新时间
}

// MaterialQuoteCostItem 是详细报价内嵌成本项，前端按类型分区后再按 index 升序展示。
type MaterialQuoteCostItem struct {
	Index        int     `json:"index" bson:"index"`                 // 成本项排序序号，前端按升序展示
	CategoryCode string  `json:"category_code" bson:"category_code"` // 成本类型编码：material、process、labor_equipment、quality、packing_logistics、management、tooling、loss、other
	CategoryName string  `json:"category_name" bson:"category_name"` // 成本类型名称：材料成本、加工工序成本、人工/设备成本等
	Name         string  `json:"name" bson:"name"`                   // 成本项名称，如原材料、折弯、焊接、打包费
	Enabled      bool    `json:"enabled" bson:"enabled"`             // 是否启用该成本项，未启用时不参与汇总
	Custom       bool    `json:"custom" bson:"custom"`               // 是否为手工新增的定制成本项
	Amount       float64 `json:"amount" bson:"amount"`               // 成本金额
	Remark       string  `json:"remark" bson:"remark"`               // 成本项备注
}
