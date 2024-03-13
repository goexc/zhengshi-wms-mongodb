package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	Id             primitive.ObjectID `json:"_id" bson:"_id"`
	Name           string             `json:"name" bson:"name"`
	Password       string             `json:"password" bson:"password"`
	Mobile         string             `json:"mobile" bson:"mobile"`
	Email          string             `json:"email" bson:"email"`
	Avatar         string             `json:"avatar" bson:"avatar"`
	Sex            string             `json:"sex" bson:"sex"`                         //性别
	DepartmentId   string             `json:"department_id" bson:"department_id"`     //部门id
	DepartmentName string             `json:"department_name" bson:"department_name"` //部门名称
	Status         string             `json:"status" bson:"status"`                   //用户状态：启用，禁用，删除
	Remark         string             `json:"remark" bson:"remark"`                   //备注
	CreatedAt      int64              `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt      int64              `json:"updated_at" bson:"updated_at"`
}
