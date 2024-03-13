package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 物料分类
type MaterialCategory struct {
	Id          primitive.ObjectID `json:"_id" bson:"_id,omitempty"`                  //
	ParentId    string             `json:"parent_id" bson:"parent_id"`                //上级物料分类id
	SortId      int64              `json:"sort_id" bson:"sort_id"`                    //排序
	Name        string             `json:"name" bson:"name"`                          //物料分类名称
	Status      string             `json:"status" bson:"status"`                      //状态：启用、停用
	Remark      string             `json:"remark" bson:"remark"`                      //备注
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at" bson:"created_at"`              //
	UpdatedAt   int64              `json:"updated_at" bson:"updated_at"`              //
}
