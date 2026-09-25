import { request, parseToObject } from "@/.cool";
import type { OutboundQuery, OutboundOrder, OutboundPage, OutboundMaterial, OutboundTotals, OutboundUnitTotal } from "@/types/outbound";

function stringField(row: UTSJSONObject, key: string): string {
	const value = row[key];
	return typeof value == "string" ? value as string : "";
}

function numberField(row: UTSJSONObject, key: string): number | null {
	const value = row[key];
	return typeof value == "number" && isFinite(value as number) ? value as number : null;
}

export function normalizeOutboundOrder(value: any): OutboundOrder {
	const row = parseToObject(value);
	return {
		id: stringField(row, "id"), code: stringField(row, "code"), status: stringField(row, "status"), type: stringField(row, "type"),
		customer_id: stringField(row, "customer_id"), customer_name: stringField(row, "customer_name"),
		supplier_id: stringField(row, "supplier_id"), supplier_name: stringField(row, "supplier_name"),
		carrier_name: stringField(row, "carrier_name"), carrier_cost: numberField(row, "carrier_cost"), other_cost: numberField(row, "other_cost"),
		total_amount: numberField(row, "total_amount"), departure_time: numberField(row, "departure_time") ?? 0,
		receipt_time: numberField(row, "receipt_time") ?? 0, remark: stringField(row, "remark")
	};
}

export function normalizeOutboundMaterial(value: any): OutboundMaterial {
	const row = parseToObject(value);
	const code = stringField(row, "order_code");
	return {
		id: stringField(row, "id"), order_code: code != "" ? code : stringField(row, "code"),
		material_id: stringField(row, "material_id"), index: numberField(row, "index") ?? 0,
		name: stringField(row, "name"), model: stringField(row, "model"), specification: stringField(row, "specification"),
		unit: stringField(row, "unit"), price: numberField(row, "price"), quantity: numberField(row, "quantity"), weight: numberField(row, "weight"),
		departure_date: numberField(row, "departure_date") ?? 0, receipt_date: numberField(row, "receipt_date") ?? 0
	};
}

export function outboundQuery(page: number = 1): OutboundQuery {
	// Omitting these fields makes the Go zero-value filter for unpacked/unweighed orders.
	return { page, size: 20, code: "", customer_id: "", model: "", status: "", type: "", is_pack: -1, is_weigh: -1 };
}

export async function getOutboundPage(query: OutboundQuery): Promise<OutboundPage> {
	const data = await request({ url: "/outbound/page", method: "GET", data: parseToObject(query) });
	if (data == null) throw new Error("出库单响应为空，请重试");
	const body = parseToObject(data);
	const total = numberField(body, "total");
	if (total == null || total < 0) throw new Error("出库单数量响应不完整");
	const rawList = body["list"];
	if (rawList != null && !Array.isArray(rawList)) throw new Error("出库单列表响应格式异常");
	const list = ((rawList ?? ([] as any[])) as any[]).map((row: any): OutboundOrder => normalizeOutboundOrder(row));
	return { total, list };
}

export async function getExactOutboundOrder(code: string): Promise<OutboundOrder | null> {
	const query = outboundQuery();
	query.code = code; query.size = 50;
	// The endpoint searches partial codes. Never use its first item as the selected order.
	while (true) {
		const result = await getOutboundPage(query);
		const found = result.list.find((row: OutboundOrder): boolean => row.code == code);
		if (found != null) return found;
		if (result.list.length == 0 || query.page * query.size >= result.total) return null;
		query.page += 1;
	}
}

function materialRows(data: any | null): OutboundMaterial[] {
	if (data != null && !Array.isArray(data)) throw new Error("出库明细响应格式异常");
	return ((data ?? ([] as any[])) as any[]).map((row: any): OutboundMaterial => normalizeOutboundMaterial(row));
}

export async function getOutboundMaterials(code: string): Promise<OutboundMaterial[]> {
	const data = await request({ url: "/outbound/materials", method: "GET", data: { order_code: code } });
	return materialRows(data).sort((a: OutboundMaterial, b: OutboundMaterial): number => a.index - b.index);
}

export async function getOutboundSummary(customerID: string, startDate: number, endDate: number): Promise<OutboundMaterial[]> {
	const data = await request({ url: "/outbound/summary", method: "GET", data: { customer_id: customerID, start_date: startDate, end_date: endDate } });
	return materialRows(data).sort((a: OutboundMaterial, b: OutboundMaterial): number => {
		if (a.receipt_date != b.receipt_date) return b.receipt_date - a.receipt_date;
		if (a.order_code != b.order_code) return a.order_code > b.order_code ? 1 : -1;
		return a.index - b.index;
	});
}

export function outboundLineAmount(row: OutboundMaterial): number | null {
	if (row.price == null || row.quantity == null) return null;
	return row.price * row.quantity;
}

export function summarizeOutbound(rows: OutboundMaterial[]): OutboundTotals {
	const codes: string[] = [];
	const units: OutboundUnitTotal[] = [];
	let amount = 0;
	let hasMissingAmount = false;
	let hasMissingQuantity = false;
	rows.forEach((row: OutboundMaterial) => {
		if (!codes.includes(row.order_code)) codes.push(row.order_code);
		const line = outboundLineAmount(row);
		if (line == null) hasMissingAmount = true;
		else amount += line;
		if (row.quantity == null) { hasMissingQuantity = true; return; }
		const unit = row.unit.trim() == "" ? "单位" : row.unit;
		const target = units.find((item: OutboundUnitTotal): boolean => item.unit == unit);
		if (target == null) units.push({ unit, quantity: row.quantity });
		else target.quantity += row.quantity;
	});
	return { orderCount: codes.length, lineCount: rows.length, amount: hasMissingAmount ? null : amount, units, hasMissingQuantity };
}

export function outboundDateTimestamp(value: string): number {
	const parts = value.split("-");
	if (parts.length != 3) return 0;
	const year = parseInt(parts[0], 10), month = parseInt(parts[1], 10), day = parseInt(parts[2], 10);
	const date = new Date(year, month - 1, day, 0, 0, 0);
	if (isNaN(date.getTime()) || date.getFullYear() != year || date.getMonth() != month - 1 || date.getDate() != day) return 0;
	return Math.floor(date.getTime() / 1000);
}
