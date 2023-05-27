package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*角色*/
type Role struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`                       // 角色名称
	ParentId  string             `json:"parent_id" bson:"parent_id"`             // 上级角色
	SortId    int64              `json:"sort_id" bson:"sort_id"`                 // 排序
	Status    int64              `json:"status" bson:"status"`                   // 状态：10.停用，20.在用
	Remark    string             `json:"remark" bson:"remark"`                   // 备注
	CreatedBy string             `json:"created_by" bson:"created_by,omitempty"` // 创建人
	CreatedAt int64              `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt int64              `json:"updated_at" bson:"updated_at"`
}
