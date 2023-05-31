package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Department struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type      int64              `json:"type" bson:"type"`                       // 部门类型：20.小组，40.部门，60.子公司，80.公司
	SortId    int64              `json:"sort_id" bson:"sort_id"`                 // 排序
	ParentId  string             `json:"parent_id" bson:"parent_id"`             // 上级部门
	Name      string             `json:"name" bson:"name"`                       // 部门名称
	Code      string             `json:"code" bson:"code"`                       // 部门编码
	Remark    string             `json:"remark" bson:"remark"`                   // 备注
	CreatedAt int64              `json:"created_at" bson:"created_at,omitempty"` //
	UpdatedAt int64              `json:"updated_at" bson:"updated_at"`           //
}
