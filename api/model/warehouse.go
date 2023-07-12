package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 仓库
type Warehouse struct {
	Id primitive.ObjectID `json:"_id" bson:"_id,omitempty"` //
	//仓库类型：
	//10.分销中心：用于存储和分发产品给零售商或分销商的中心仓库。
	//20.生产仓库：用于存储原材料、零部件和成品的仓库，供生产线使用。
	//30.跨境仓库：位于国际边境或海港附近，用于处理跨国贸易的仓库。
	//40.电商仓库：专门用于电子商务业务的仓库，处理在线销售订单和配送商品。
	//50.冷链仓库：具备温度控制设备和环境，用于存储和配送需要冷藏或冷冻的商品。
	//60.合规仓库：符合特定行业或监管要求的仓库，如医药品仓库、化学品仓库等。
	//70.专用仓库：根据特定产品或物品的需求而设计和定制的仓库，如危险品仓库、高值物品仓库等。
	//80.跨渠道仓库：支持多渠道销售和配送的仓库，如同时服务零售、批发和电商渠道的仓库。
	//90.自动化仓库：采用自动化设备和系统进行货物存储、搬运和管理的仓库。
	//100.第三方物流仓库：由第三方物流服务提供商经营和管理的仓库，为客户提供物流解决方案。
	Type         int     `json:"type" bson:"type"`                   // 仓库类型
	Name         string  `json:"name" bson:"name"`                   // 仓库名称
	Code         string  `json:"code" bson:"code"`                   // 仓库编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
	Address      string  `json:"address" bson:"address"`             // 仓库地址
	Capacity     float64 `json:"capacity" bson:"capacity"`           // 仓库容量
	CapacityUnit string  `json:"capacity_unit" bson:"capacity_unit"` // 仓库容量单位：面积、体积或其他度量单位
	//仓库状态
	//10.激活（Active）：表示仓库处于可用状态，可以执行库存管理和操作。
	//20.禁用（Disabled）：表示仓库处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用仓库。
	//30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的仓库。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
	//90.关闭（Closed）：表示货架已经关闭，不再进行任何库存管理和操作。通常是由于货架不再使用或被替代。
	//100.删除（Deleted）
	Status      int                `json:"status" bson:"status"`                      //仓库状态
	Manager     string             `json:"manager" bson:"manager"`                    //负责人
	Contact     string             `json:"contact" bson:"contact"`                    //联系方式
	Image       string             `json:"image" bson:"image"`                        //图片
	Remark      string             `json:"remark" bson:"remark"`                      //备注
	Creator     primitive.ObjectID `json:"creator" bson:"creator"`                    //创建人
	CreatorName string             `json:"creator_name,optional" bson:"creator_name"` //创建人
	CreatedAt   int64              `json:"created_at" bson:"created_at"`              //
	UpdatedAt   int64              `json:"updated_at" bson:"updated_at"`              //
}
