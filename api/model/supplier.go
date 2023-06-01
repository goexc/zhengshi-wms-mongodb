package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Supplier struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Name    string             `json:"name" bson:"name"`              //供应商名称
	Address string             `json:"address" bson:"address"`        //供应商地址
	Contact string             `json:"contact" bson:"contact"`        //联系方式
	Manager string             `json:"manager" bson:"manager"`        //负责人
	Level   int                `json:"level" bson:"level"`            //供应商等级
	Remark  string             `json:"remark,optional" bson:"remark"` //备注
	//以下是一些常见的供应商状态示例：
	//10.待审核（Pending Approval）：表示供应商提交了注册或变更信息，但尚未通过审核，需要系统管理员或相关人员进行审核和确认。
	//20.审核不通过（Approval Rejected）：表示供应商的注册或变更信息未通过审核，可能存在问题或不符合要求，需要供应商进行修正或重新提交。
	//30.活动（Active）：表示供应商当前处于正常状态，可以与其进行业务交互和合作。
	//40.停用（Inactive）：表示供应商被停用或暂时无法使用，可能是由于某种原因导致无法继续合作或交互。
	//50.黑名单（Blacklisted）：表示供应商因违规行为或其他原因被列入黑名单，系统会限制与该供应商的交互或合作。
	//60.合同到期（Contract Expired）：表示供应商的合同已到期，需要进行续签或重新协商合同条款。
	//100.删除(Deleted)
	Status      int                `json:"status" bson:"status"`                      //状态：10.待审核；100.删除
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at,optional" bson:"created_at"`
	UpdatedAt   int64              `json:"updated_at,optional" bson:"updated_at"`
}
