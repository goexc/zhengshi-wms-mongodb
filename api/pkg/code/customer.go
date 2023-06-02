package code

// 3.客户状态值转换
func CustomerStatusText(status int) (text string) {
	switch status {
	case 10: //10.潜在（Potential）：表示潜在的客户，即尚未正式成为系统中的活跃客户，但有潜在的合作机会。
		text = "潜在"
	case 20: //20.活动（Active）：表示客户是当前活跃的，可以进行订单处理和交互。
		text = "活动"
	case 30: //30.停用（Inactive）：表示客户暂时停止使用或被禁止使用。这可能是由于付款问题、违反合同条款、暂停业务等原因导致的。
		text = "停用"
	case 40: //40.冻结（Frozen）：表示客户的帐户被临时冻结，通常是由于安全问题、付款问题或其他问题导致的。
		text = "冻结"
	case 50: //50.黑名单（Blacklisted）：表示客户因违规行为或其他原因被列入黑名单，系统会限制与该客户的交互或合作。
		text = "黑名单"
	case 60: //60.合同到期（Contract Expired）：表示客户的合同已到期，需要进行续签或重新协商合同条款。
		text = "合同到期"
	case 100: //100.删除(Deleted)
		text = "删除"
	default:
		text = "未知"
	}
	return
}

// 4.客户状态值转换
func CustomerStatusCode(status string) (code int) {
	switch status {
	case "potential": //10.潜在（Potential）：表示潜在的客户，即尚未正式成为系统中的活跃客户，但有潜在的合作机会。
		code = 10
	case "active": //20.活动（Active）：表示客户是当前活跃的，可以进行订单处理和交互。
		code = 20
	case "inactive": //30.停用（Inactive）：表示客户暂时停止使用或被禁止使用。这可能是由于付款问题、违反合同条款、暂停业务等原因导致的。
		code = 30
	case "frozen": //40.冻结（Frozen）：表示客户的帐户被临时冻结，通常是由于安全问题、付款问题或其他问题导致的。
		code = 40
	case "blacklisted": //50.黑名单（Blacklisted）：表示客户因违规行为或其他原因被列入黑名单，系统会限制与该客户的交互或合作。
		code = 50
	case "contract_expired": //60.合同到期（Contract Expired）：表示客户的合同已到期，需要进行续签或重新协商合同条款。
		code = 60
	case "deleted": //100.删除(Deleted)
		code = 100
	default: //未知状态
		code = 0
	}
	return
}
