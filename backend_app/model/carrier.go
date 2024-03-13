package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Carrier struct {
	Id                            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type                          string             `json:"type" bson:"type"`                                                         //承运商类型：个人、企业、组织
	Name                          string             `json:"name" bson:"name"`                                                         //承运商名称
	Code                          string             `json:"code" bson:"code"`                                                         //承运商编号：分配给承运商的唯一标识符或编号，用于快速识别和检索客户信息
	Image                         string             `json:"image" bson:"image"`                                                       //承运商图片
	LegalRepresentative           string             `json:"legal_representative" bson:"legal_representative"`                         //法定代表人
	UnifiedSocialCreditIdentifier string             `json:"unified_social_credit_identifier" bson:"unified_social_credit_identifier"` //统一社会信用代码
	Address                       string             `json:"address" bson:"address"`                                                   //承运商地址
	Manager                       string             `json:"manager" bson:"manager"`                                                   //负责人
	Contact                       string             `json:"contact" bson:"contact"`                                                   //联系方式
	Email                         string             `json:"email" bson:"email"`                                                       //Email
	//以下是一些常见的承运商状态示例：
	//10.审核中（Pending Approval）：表示承运商提交了注册或变更信息，但尚未通过审核，需要系统管理员或相关人员进行审核和确认。
	//20.审核不通过（Approval Rejected）：表示承运商的注册或变更信息未通过审核，可能存在问题或不符合要求，需要承运商进行修正或重新提交。
	//30.活动（Active）：表示承运商当前处于正常状态，可以与其进行业务交互和合作。
	//40.停用（Inactive）：表示承运商被停用或暂时无法使用，可能是由于某种原因导致无法继续合作或交互。
	//50.黑名单（Blacklisted）：表示承运商因违规行为或其他原因被列入黑名单，系统会限制与该承运商的交互或合作。
	//60.合同到期（Contract Expired）：表示承运商的合同已到期，需要进行续签或重新协商合同条款。
	//100.删除(Deleted)
	Status      string             `json:"status" bson:"status"`                      //状态
	Remark      string             `json:"remark,optional" bson:"remark"`             //备注
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at,optional" bson:"created_at"`
	UpdatedAt   int64              `json:"updated_at,optional" bson:"updated_at"`
}
