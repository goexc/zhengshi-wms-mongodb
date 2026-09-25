import { request, parseToObject } from "@/.cool";
import type { CustomerChoice } from "@/types/outbound";

export async function findCustomerChoices(name: string): Promise<CustomerChoice[]> {
	const response = await request({ url: "/customer/list", method: "GET", data: { name: name.trim() } });
	if (response == null) throw new Error("客户列表响应为空");
	const data = parseToObject(response);
	const list = data["list"];
	if (list != null && !Array.isArray(list)) throw new Error("客户列表响应格式异常");
	const result: CustomerChoice[] = [];
	((list ?? ([] as any[])) as any[]).forEach((value: any) => {
		const row = parseToObject(value);
		const id = `${row["id"] ?? ""}`, name = `${row["name"] ?? ""}`;
		if (id != "") result.push({ id, name, code: `${row["code"] ?? ""}` });
	});
	return result;
}
