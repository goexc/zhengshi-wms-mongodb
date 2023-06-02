package code

// 5.库区状态值转换明文
func WarehouseZoneStatusText(status int) (text string) {
	switch status {
	case 10: //10.激活（Active）：表示库区处于可用状态，可以执行库存管理和操作。
		text = "激活"
	case 20: //20.禁用（Disabled）：表示库区处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用库区。
		text = "禁用"
	case 30: //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的库区。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		text = "盘点中"
	case 90: //90.关闭（Closed）：表示库区已经关闭，不再进行任何库存管理和操作。通常是由于库区不再使用或被替代。
		text = "关闭"
	case 100: //100.删除（Deleted）
		text = "删除"
	default: //未知状态
		text = "未知状态"
	}
	return
}

// 6.库区状态明文转换值
func WarehouseZoneStatusCode(text string) (code int) {
	switch text {
	case "激活": //10.激活（Active）：表示库区处于可用状态，可以执行库存管理和操作。
		code = 10
	case "禁用": //20.禁用（Disabled）：表示库区处于禁用状态，不可用于库存管理和操作。通常是由于维护、修复或其他原因暂时停用库区。
		code = 20
	case "盘点中": //30.盘点中(Stocktake )：用于表示当前正在进行盘点活动的库区。这样可以确保在盘点期间，其他库存管理操作不会干扰盘点过程。
		code = 30
	case "关闭": //90.关闭（Closed）：表示库区已经关闭，不再进行任何库存管理和操作。通常是由于库区不再使用或被替代。
		code = 90
	case "删除": //100.删除（Deleted）
		code = 100
	default: //未知状态
		code = 0
	}
	return
}
