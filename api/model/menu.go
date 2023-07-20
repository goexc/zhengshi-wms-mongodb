package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Menu struct {
	Id         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type       int64              `json:"type" bson:"type"`                       //路由类型：1.菜单，2.按钮
	Name       string             `json:"name" bson:"name"`                       //路由名称：如，System
	Path       string             `json:"path" bson:"path"`                       //路由路径
	ParentId   string             `json:"parent_id" bson:"parent_id"`             //父路由id
	SortId     int64              `json:"sort_id" bson:"sort_id"`                 //排序
	Component  string             `json:"component" bson:"component"`             //路由组件
	Title      string             `json:"title" bson:"title"`                     //标题
	Icon       string             `json:"icon" bson:"icon"`                       //图标
	Transition string             `json:"transition" bson:"transition"`           //过渡动画
	Hidden     bool               `json:"hidden" bson:"hidden"`                   //是否隐藏
	Fixed      bool               `json:"fixed" bson:"fixed"`                     //是否固定
	IsFull     bool               `json:"is_full" bson:"is_full"`                 //是否全屏
	Perms      string             `json:"perms" bson:"perms"`                     //权限标识
	Remark     string             `json:"remark" bson:"remark"`                   //备注
	CreatedAt  int64              `json:"created_at" bson:"created_at,omitempty"` //
	UpdatedAt  int64              `json:"updated_at" bson:"updated_at"`           //
}
