package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// MaterialDeliveryRebuildTask records one customer new-material rebuild run.
type MaterialDeliveryRebuildTask struct {
	Id            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`             // 任务id
	Status        string             `json:"status" bson:"status"`                 // 任务状态：queued、running、success、failed
	OrderCount    int64              `json:"order_count" bson:"order_count"`       // 已扫描出库单数量
	DeliveryCount int64              `json:"delivery_count" bson:"delivery_count"` // 已生成或更新的客户新增物料记录数量
	Message       string             `json:"message" bson:"message"`               // 任务执行消息
	ErrorMessage  string             `json:"error_message" bson:"error_message"`   // 失败原因
	CreatorId     string             `json:"creator_id" bson:"creator_id"`         // 创建人id
	CreatorName   string             `json:"creator_name" bson:"creator_name"`     // 创建人名称
	CreatedAt     int64              `json:"created_at" bson:"created_at"`         // 创建时间
	StartedAt     int64              `json:"started_at" bson:"started_at"`         // 开始执行时间
	FinishedAt    int64              `json:"finished_at" bson:"finished_at"`       // 结束时间
	UpdatedAt     int64              `json:"updated_at" bson:"updated_at"`         // 更新时间
}
