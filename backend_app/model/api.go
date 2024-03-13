package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Api struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type      int64              `json:"type" bson:"type"`           // 类型：1.模块，2.API
	SortId    int64              `json:"sort_id" bson:"sort_id"`     // 排序
	ParentId  string             `json:"parent_id" bson:"parent_id"` // 上级id
	Uri       string             `json:"uri" bson:"uri"`             // 请求路径
	Method    string             `json:"method" bson:"method"`       // 请求方法
	Name      string             `json:"name" bson:"name"`           // 名称
	Remark    string             `json:"remark" bson:"remark"`       // 备注
	CreatedAt int64              `json:"created_at" bson:"created_at"`
	UpdatedAt int64              `json:"updated_at" bson:"updated_at"`
}
