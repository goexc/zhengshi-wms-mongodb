import { parse, parseToObject, request } from "@/.cool";
import type { CustomerTransaction, CustomerTransactionPage, PartnerPage, PartnerQuery } from "@/types/business";

export async function fetchPartners(kind: string, query: PartnerQuery): Promise<PartnerPage> {
	if (kind != "customer" && kind != "supplier") throw new Error("业务对象类型无效");
	const data = await request({ url: kind == "customer" ? "/customer" : "/supplier", method: "GET", data: parseToObject(query) });
	const result = parse<PartnerPage>(data);
	if (result == null) throw new Error("企业资料响应为空，请重试");
	return { total: result.total ?? 0, list: result.list ?? [] } as PartnerPage;
}

export async function fetchCustomerTransactions(customerId: string, page: number): Promise<CustomerTransactionPage> {
	if (!/^[a-fA-F0-9]{24}$/.test(customerId)) throw new Error("客户信息不完整，请返回客户列表重新选择");
	const data = await request({ url: "/customer/transaction", method: "GET", data: { customer_id: customerId, page, size: 20 } });
	const result = parse<CustomerTransactionPage>(data);
	if (result == null) throw new Error("客户流水响应为空，请重试");
	return { total: result.total ?? 0, list: result.list ?? [] } as CustomerTransactionPage;
}

export function transactionDirection(item: CustomerTransaction): string {
	if (item.direction == "receivable_increase") return "应收增加";
	if (item.direction == "receivable_decrease") return "应收减少";
	if ((item.direction ?? "") != "") return "方向未识别";
	// 兼容服务端财务契约明确支持的历史记录；调整类不能猜测方向。
	if (item.transaction_type == "opening_ar" || item.transaction_type == "outbound_ar" || item.type == "期初应收" || item.type == "应收账款") return "应收增加";
	if (item.transaction_type == "payment" || item.transaction_type == "return_credit" || item.type == "回款" || item.type == "退货") return "应收减少";
	return "方向未记录";
}

export function transactionStatus(status: string): string {
	if (status == "confirmed") return "已确认";
	if (status == "draft") return "草稿 · 不计入余额";
	if (status == "reversed") return "已冲销 · 不计入余额";
	if (status == "voided") return "已作废 · 不计入余额";
	return status == "" ? "历史记录" : status;
}

export function transactionSource(source: string): string {
	if (source == "opening") return "期初应收";
	if (source == "outbound_order") return "出库单";
	if (source == "inbound_return") return "退货入库";
	if (source == "payment") return "回款";
	if (source == "manual_adjustment") return "手工调整";
	if (source == "system_recount") return "系统重算";
	return source == "" ? "未记录" : source;
}
