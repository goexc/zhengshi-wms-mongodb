package code

// 1.入库单状态值转换明文
func InboundReceiptStatusText(status int) (text string) {
	switch status {
	case 10: //10.待审核：入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
		text = "待审核"
	case 20: //20.审核不通过：入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
		text = "审核不通过"
	case 30: //30.审核通过：入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
		text = "审核通过"
	case 40: //40.未发货：
		text = "未发货"
	//case 50: //50.在途：
	//	text = "在途"
	case 60: //60.部分入库：当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
		text = "部分入库"
	case 70: //70.作废：当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
		text = "作废"
	case 80: //80.入库完成：当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
		text = "入库完成"
	default: //未知状态
		text = "未知状态"
	}
	return
}

// 2.入库单状态明文转换值
func InboundReceiptStatusCode(text string) (code int) {
	switch text {
	case "待审核": //10.待审核：入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
		code = 10
	case "审核不通过": //20.审核不通过：入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
		code = 20
	case "审核通过": //30.审核通过：入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
		code = 30
	case "未发货": //40.未发货：
		code = 40
	//case "在途": //50.在途：
	//	code = 50
	case "部分入库": //60.部分入库：当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
		code = 60
	case "作废": //70.作废：当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
		code = 70
	case "入库完成": //80.入库完成：当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
		code = 80
	default: //未知状态
		code = 0
	}
	return
}

// 3.根据物料状态判断入库单状态
func Material2InboundReceiptStatus(statuses map[string]int) (status string) {
	//1.空map
	if len(statuses) == 0 {
		return
	}

	//2.只有一种状态
	if len(statuses) == 1 {
		for key := range statuses {
			return key
		}
	}

	//3.状态排序：忽略"作废"状态
	//var _s = []string{"入库完成", "部分入库", "在途", "未发货"}
	var _s = []string{"部分入库", "未发货"}

	//3.返回状态
	for _, one := range _s {
		if _, ok := statuses[one]; ok {
			return one
		}
	}

	return
}
