import { parse, request } from "@/.cool";
import type { PlanPage } from "@/types/plan";

export async function fetchPlans(page: number, status: string): Promise<PlanPage> {
	const data = await request({ url: "/plan", method: "GET", data: { page, size: 20, status } });
	const result = parse<PlanPage>(data);
	if (result == null) throw new Error("计划响应为空，请重试");
	if (result.total == null || !Number.isFinite(result.total) || result.total < 0) {
		throw new Error("计划分页总数异常，请重试");
	}
	return { total: result.total, list: result.list ?? [] } as PlanPage;
}
