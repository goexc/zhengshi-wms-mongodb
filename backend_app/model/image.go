package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Image struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	ObjectKey string             `json:"object_key" bson:"object_key"`
	Size      int64              `json:"size" bson:"size"` //文件大小：KB
	CreatedAt int64              `json:"created_at" bson:"created_at,omitempty"`
}
