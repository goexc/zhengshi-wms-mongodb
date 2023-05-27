package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id"`
	Account   string             `json:"account" bson:"account"`
	Password  string             `json:"password" bson:"password"`
	Mobile    string             `json:"mobile" bson:"mobile"`
	Email     string             `json:"email" bson:"email"`
	Avatar    string             `json:"avatar" bson:"avatar"`
	Sex       int                `json:"sex" bson:"sex"`       //性别：0.女，1.男，2.未知
	Status    int                `json:"status" bson:"status"` //用户状态：0.未启用，20.启用，50.禁用
	CreatedAt int64              `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt int64              `json:"updated_at" bson:"updated_at"`
}
