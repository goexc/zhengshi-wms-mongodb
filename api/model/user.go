package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	Id             primitive.ObjectID `json:"_id" bson:"_id"`
	Account        string             `json:"account" bson:"account"`
	Password       string             `json:"password" bson:"password"`
	Mobile         string             `json:"mobile" bson:"mobile"`
	Email          string             `json:"email" bson:"email"`
	Avatar         string             `json:"avatar" bson:"avatar"`
	Sex            string             `json:"sex" bson:"sex"`       //性别
	DepartmentId   string             `json:"department_id"`        //部门id
	DepartmentName string             `json:"department_name"`      //部门名称
	Status         int                `json:"status" bson:"status"` //用户状态：0.未启用，20.启用，50.禁用
	Remark         string             `json:"remark"`               //备注
	CreatedAt      int64              `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt      int64              `json:"updated_at" bson:"updated_at"`
}
