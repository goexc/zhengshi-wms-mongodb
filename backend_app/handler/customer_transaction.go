package handler

import (
	"backend/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// CustomerTransactionSync is retained only so old migration callers fail
// closed. Rebuilding receivables from order codes would overwrite adjustments
// and can recreate finance records after an outbound order has been removed.
func CustomerTransactionSync(_ *svc.ServiceContext) {
	logx.Error("旧客户流水同步已停用：请使用 API 客户余额重算，禁止按出库单号覆盖客户流水")
}
