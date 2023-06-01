package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Customer struct {
	Id                            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Name                          string             `json:"name" bson:"name"`                                                         //客户名称
	Type                          int                `json:"type" bson:"type"`                                                         //客户类型：10.个人、20.企业、30.组织
	Code                          string             `json:"code" bson:"code"`                                                         //客户编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
	LegalRepresentative           string             `json:"legal_representative" bson:"legal_representative"`                         //法定代表人
	UnifiedSocialCreditIdentifier string             `json:"unified_social_credit_identifier" bson:"unified_social_credit_identifier"` //统一社会信用代码
	Address                       string             `json:"address" bson:"address"`                                                   //客户地址
	Contact                       string             `json:"contact" bson:"contact"`                                                   //联系方式
	Manager                       string             `json:"manager" bson:"manager"`                                                   //负责人
	Email                         string             `json:"email" bson:"email"`                                                       //Email
	Level                         int                `json:"level" bson:"level"`                                                       //客户等级
	Remark                        string             `json:"remark,optional" bson:"remark"`                                            //备注
	//以下是一些常见的客户状态示例：
	//10.潜在（Potential）：表示潜在的客户，即尚未正式成为系统中的活跃客户，但有潜在的合作机会。
	//20.活动（Active）：表示客户是当前活跃的，可以进行订单处理和交互。
	//30.停用（Inactive）：表示客户暂时停止使用或被禁止使用。这可能是由于付款问题、违反合同条款、暂停业务等原因导致的。
	//40.冻结（Frozen）：表示客户的帐户被临时冻结，通常是由于安全问题、付款问题或其他问题导致的。
	//50.黑名单（Blacklisted）：表示客户因违规行为或其他原因被列入黑名单，系统会限制与该客户的交互或合作。
	//60.合同到期（Contract Expired）：表示客户的合同已到期，需要进行续签或重新协商合同条款。
	//100.删除(Deleted)
	Status      int                `json:"status" bson:"status"`                      //状态
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at,optional" bson:"created_at"`
	UpdatedAt   int64              `json:"updated_at,optional" bson:"updated_at"`
}
