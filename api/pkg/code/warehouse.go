package code

// 1.仓库类型值转换明文
func WarehouseTypeText(t int) (text string) {
	switch t {
	case 10: //10.分销中心：用于存储和分发产品给零售商或分销商的中心仓库。
		text = "分销中心"
	case 20: //20.生产仓库：用于存储物料、零部件和成品的仓库，供生产线使用。
		text = "生产仓库"
	case 30: //30.跨境仓库：位于国际边境或海港附近，用于处理跨国贸易的仓库。
		text = "跨境仓库"
	case 40: //40.电商仓库：专门用于电子商务业务的仓库，处理在线销售订单和配送商品。
		text = "电商仓库"
	case 50: //50.冷链仓库：具备温度控制设备和环境，用于存储和配送需要冷藏或冷冻的商品。
		text = "冷链仓库"
	case 60: //60.合规仓库：符合特定行业或监管要求的仓库，如医药品仓库、化学品仓库等。
		text = "合规仓库"
	case 70: //70.专用仓库：根据特定产品或物品的需求而设计和定制的仓库，如危险品仓库、高值物品仓库等。
		text = "专用仓库"
	case 80: //80.跨渠道仓库：支持多渠道销售和配送的仓库，如同时服务零售、批发和电商渠道的仓库。
		text = "跨渠道仓库"
	case 90: //90.自动化仓库：采用自动化设备和系统进行货物存储、搬运和管理的仓库。
		text = "自动化仓库"
	case 100: //100.第三方物流仓库：由第三方物流服务提供商经营和管理的仓库，为客户提供物流解决方案。
		text = "第三方物流仓库"
	default: //未知类型
		text = "未知类型"
	}
	return
}

// 2.仓库类型明文转换值
func WarehouseTypeCode(text string) (code int) {
	switch text {
	case "分销中心": //10.分销中心：用于存储和分发产品给零售商或分销商的中心仓库。
		code = 10
	case "生产仓库": //20.生产仓库：用于存储物料、零部件和成品的仓库，供生产线使用。
		code = 20
	case "跨境仓库": //30.跨境仓库：位于国际边境或海港附近，用于处理跨国贸易的仓库。
		code = 30
	case "电商仓库": //40.电商仓库：专门用于电子商务业务的仓库，处理在线销售订单和配送商品。
		code = 40
	case "冷链仓库": //50.冷链仓库：具备温度控制设备和环境，用于存储和配送需要冷藏或冷冻的商品。
		code = 50
	case "合规仓库": //60.合规仓库：符合特定行业或监管要求的仓库，如医药品仓库、化学品仓库等。
		code = 60
	case "专用仓库": //70.专用仓库：根据特定产品或物品的需求而设计和定制的仓库，如危险品仓库、高值物品仓库等。
		code = 70
	case "跨渠道仓库": //80.跨渠道仓库：支持多渠道销售和配送的仓库，如同时服务零售、批发和电商渠道的仓库。
		code = 80
	case "自动化仓库": //90.自动化仓库：采用自动化设备和系统进行货物存储、搬运和管理的仓库。
		code = 90
	case "第三方物流仓库": //100.第三方物流仓库：由第三方物流服务提供商经营和管理的仓库，为客户提供物流解决方案。
		code = 100
	default: //未知类型
		code = 0
	}
	return
}

// 3.仓库状态值转换明文
func WarehouseStatusText(status int) (text string) {
	switch status {
	case 10: //10.激活（Active）：表示仓库处于可用状态，可以执行库存管理和操作。
		text = "激活"
	case 20: //20.禁用（Disabled）：表示仓库处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用仓库。
		text = "禁用"
	case 30: //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的仓库。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		text = "盘点中"
	case 90: //90.关闭（Closed）：表示仓库已经关闭，不再进行任何库存管理和操作。通常是由于仓库不再使用或被替代。
		text = "关闭"
	case 100: //100.删除（Deleted）
		text = "删除"
	default: //未知状态
		text = "未知状态"
	}
	return
}

// 4.仓库状态明文转换值
func WarehouseStatusCode(text string) (code int) {
	switch text {
	case "激活": //10.激活（Active）：表示仓库处于可用状态，可以执行库存管理和操作。
		code = 10
	case "禁用": //20.禁用（Disabled）：表示仓库处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用仓库。
		code = 20
	case "盘点中": //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的仓库。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		code = 30
	case "关闭": //90.关闭（Closed）：表示仓库已经关闭，不再进行任何库存管理和操作。通常是由于仓库不再使用或被替代。
		code = 90
	case "删除": //100.删除（Deleted）
		code = 100
	default: //未知状态
		code = 0
	}
	return
}
