package pkg

// 1.供应商状态值转换
func SupplierStatusText(status int) (text string) {
	switch status {
	case 10: //10.待审核（Pending Approval）：表示供应商提交了注册或变更信息，但尚未通过审核，需要系统管理员或相关人员进行审核和确认。
		text = "待审核"
	case 20: //20.审核不通过（Approval Rejected）：表示供应商的注册或变更信息未通过审核，可能存在问题或不符合要求，需要供应商进行修正或重新提交。
		text = "审核不通过"
	case 30: //30.活动（Active）：表示供应商当前处于正常状态，可以与其进行业务交互和合作。
		text = "活动"
	case 40: //40.停用（Inactive）：表示供应商被停用或暂时无法使用，可能是由于某种原因导致无法继续合作或交互。
		text = "停用"
	case 50: //50.黑名单（Blacklisted）：表示供应商因违规行为或其他原因被列入黑名单，系统会限制与该供应商的交互或合作。
		text = "黑名单"
	case 60: //60.合同到期（Contract Expired）：表示供应商的合同已到期，需要进行续签或重新协商合同条款。
		text = "合同到期"
	case 100: //100.删除(Deleted)
		text = "删除"
	default:
		text = "未知"
	}
	return
}

// 2.供应商状态值转换
func SupplierStatusCode(status string) (code int) {
	switch status {
	case "pending_approval": //10.待审核（Pending Approval）：表示供应商提交了注册或变更信息，但尚未通过审核，需要系统管理员或相关人员进行审核和确认。
		code = 10
	case "approval_rejected": //20.审核不通过（Approval Rejected）：表示供应商的注册或变更信息未通过审核，可能存在问题或不符合要求，需要供应商进行修正或重新提交。
		code = 20
	case "active": //30.活动（Active）：表示供应商当前处于正常状态，可以与其进行业务交互和合作。
		code = 30
	case "inactive": //40.停用（Inactive）：表示供应商被停用或暂时无法使用，可能是由于某种原因导致无法继续合作或交互。
		code = 40
	case "blacklisted": //50.黑名单（Blacklisted）：表示供应商因违规行为或其他原因被列入黑名单，系统会限制与该供应商的交互或合作。
		code = 50
	case "contract_expired": //60.合同到期（Contract Expired）：表示供应商的合同已到期，需要进行续签或重新协商合同条款。
		code = 60
	case "deleted": //100.删除(Deleted)
		code = 100
	default: //未知状态
		code = 0
	}
	return
}
