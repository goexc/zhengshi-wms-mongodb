package code

// 1.货架类型值转换明文
func WarehouseRackTypeText(t int) (text string) {
	switch t {
	//10.标准货架 - Standard Shelf
	//20.重型货架 - Heavy-duty Shelf
	//30.中型货架 - Medium-duty Shelf
	//40.轻型货架 - Light-duty Shelf
	case 10:
		text = "标准货架"
	case 20:
		text = "重型货架"
	case 30:
		text = "中型货架"
	case 40:
		text = "轻型货架"
	default: //未知类型
		text = "未知类型货架"
	}
	return
}

// 2.货架类型明文转换值
func WarehouseRackTypeCode(text string) (code int) {
	switch text {
	case "标准货架":
		code = 10
	case "重型货架":
		code = 20
	case "中型货架":
		code = 30
	case "轻型货架":
		code = 40
	default: //未知类型
		code = 0
	}
	return
}

// 3.货架状态值转换明文
func WarehouseRackStatusText(status int) (text string) {
	switch status {
	case 10: //10.激活（Active）：表示货架处于可用状态，可以执行库存管理和操作。
		text = "激活"
	case 20: //20.禁用（Disabled）：表示货架处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用货架。
		text = "禁用"
	case 30: //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的货架。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		text = "盘点中"
	case 90: //90.关闭（Closed）：表示货架已经关闭，不再进行任何库存管理和操作。通常是由于货架不再使用或被替代。
		text = "关闭"
	case 100: //100.删除（Deleted）
		text = "删除"
	default: //未知状态
		text = "未知状态"
	}
	return
}

// 4.货架状态明文转换值
func WarehouseRackStatusCode(text string) (code int) {
	switch text {
	case "激活": //10.激活（Active）：表示货架处于可用状态，可以执行库存管理和操作。
		code = 10
	case "禁用": //20.禁用（Disabled）：表示货架处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用货架。
		code = 20
	case "盘点中": //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的货架。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		code = 30
	case "关闭": //90.关闭（Closed）：表示货架已经关闭，不再进行任何库存管理和操作。通常是由于货架不再使用或被替代。
		code = 90
	case "删除": //100.删除（Deleted）
		code = 100
	default: //未知状态
		code = 0
	}
	return
}
