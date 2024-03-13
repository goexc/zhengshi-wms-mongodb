package code

// 出库单状态转范围列表
func OutboundStatusRange(status string) []string {
	switch status {
	case "预发货": //出库单尚未确认
		return []string{"预发货"}

	case "待拣货": //出库单确认后的状态
		return []string{"待拣货"}

	case "已拣货": //出库单物料已拣货
		return []string{"已拣货"}

	case "待打包": //已拣货、已称重维达堡的出库单可以选择打包
		return []string{"已拣货", "已称重"}

	case "已打包": //出库单已执行打包操作
		return []string{"已打包"}

	case "待称重": //已拣货、已打包围城中的出库单可以称重
		return []string{"已拣货", "已打包"}

	case "已称重": //出库单已称重
		return []string{"已称重"}

	case "待出库": //已拣货、已打包、已称重的出库单可以出库
		return []string{"已拣货", "已打包", "已称重"}

	case "已出库": //出库单已出库
		return []string{"已出库"}

	case "已签收": //出库单已签收
		return []string{"已签收"}

	default: //查询全部
		return nil
	}
}
