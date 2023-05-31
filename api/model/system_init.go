package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SystemInit struct {
	Id   primitive.ObjectID `json:"_id" bson:"_id,omitempty"` //
	Step int                `json:"step" bson:"step"`         // 系统初始化步骤:
	// -1.系统初始化失败，是否重新初始化；
	// 0.系统未初始化，写入API信息；
	// 1.此步骤写入菜单信息；
	// 2.此步骤写入超级管理员角色；
	// 3.此步骤填写企业信息
	// 4.此步骤填写运维部门信息
	// 5.此步骤设置系统管理员密码
	// 6.系统初始化完毕
	CreatedAt int64 `json:"created_at" bson:"created_at"` //
	UpdatedAt int64 `json:"updated_at" bson:"updated_at"` //
}
