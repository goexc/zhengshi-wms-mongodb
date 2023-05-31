package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Company struct {
	Id                            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Name                          string             `json:"name" bson:"name"`                                                         //
	Address                       string             `json:"address" bson:"address"`                                                   //
	Contact                       string             `json:"contact" bson:"contact"`                                                   //联系方式
	LegalRepresentative           string             `json:"legal_representative" bson:"legal_representative"`                         //法定代表人
	UnifiedSocialCreditIdentifier string             `json:"unified_social_credit_identifier" bson:"unified_social_credit_identifier"` //统一社会信用代码
	Email                         string             `json:"email" bson:"email"`                                                       //Email
	Site                          string             `json:"site" bson:"site"`                                                         //企业网址
	Introduction                  string             `json:"introduction" bson:"introduction"`                                         //简介
	BusinessScope                 string             `json:"business_scope" bson:"business_scope"`                                     //经营范围
	CreatedAt                     int64              `json:"created_at" bson:"created_at,omitempty"`                                   //
	UpdatedAt                     int64              `json:"updated_at" bson:"updated_at"`                                             //
}
