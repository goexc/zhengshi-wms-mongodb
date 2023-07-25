package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Material struct {
	Id               primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Name             string             `json:"name" bson:"name"`                           //物料名称
	CategoryId       string             `json:"category_id" bson:"category_id"`             //物料分类Id
	CategoryName     string             `json:"category_name" bson:"category_name"`         //物料分类名称
	Image            string             `json:"image" bson:"image"`                         //物料图片
	Model            string             `json:"model" bson:"model"`                         //型号：用于唯一标识和区分不同种类的钢材。
	Material         string             `json:"material" bson:"material"`                   //材质：碳钢、不锈钢、合金钢等。
	Specification    string             `json:"specification" bson:"specification"`         //规格：包括长度、宽度、厚度等尺寸信息。
	SurfaceTreatment string             `json:"surface_treatment" bson:"surface_treatment"` //表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
	StrengthGrade    string             `json:"strength_grade" bson:"strength_grade"`       //强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
	Unit             string             `json:"unit" bson:"unit"`                           //计量单位
	Remark           string             `json:"remark" bson:"remark"`                       //备注
	Creator          primitive.ObjectID `json:"creator" bson:"creator"`                     //创建人
	CreatorName      string             `json:"creator_name,optional" bson:"creator_name"`  //创建人
	CreatedAt        int64              `json:"created_at" bson:"created_at"`
	UpdatedAt        int64              `json:"updated_at" bson:"updated_at"`
}
