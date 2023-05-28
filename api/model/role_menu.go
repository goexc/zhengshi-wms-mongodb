package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*角色菜单集合*/
type RoleMenu struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	RoleId    string             `json:"role_id" bson:"role_id"` //
	MenuId    string             `json:"menu_id" bson:"menu_id"` //
	CreatedAt int64              `json:"created_at" bson:"created_at,omitempty"`
}
