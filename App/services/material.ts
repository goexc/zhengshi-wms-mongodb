import { parse, parseToObject, request } from "@/.cool";
import type {
	Material, MaterialCategory, MaterialCategoryListPayload, MaterialPage,
	MaterialPagePayload, MaterialPageRequest,
	MaterialRequest
} from "@/types/material";

// WMS info omits prices; list responses include customer prices.
function normalizeMaterial(item: Material): Material {
	item.id = item.id ?? "";
	item.image = item.image ?? "";
	item.category_id = item.category_id ?? "";
	item.category_name = item.category_name ?? "";
	item.name = item.name ?? "";
	item.material = item.material ?? "";
	item.specification = item.specification ?? "";
	item.model = item.model ?? "";
	item.surface_treatment = item.surface_treatment ?? "";
	item.strength_grade = item.strength_grade ?? "";
	item.unit = item.unit ?? "";
	item.remark = item.remark ?? "";
	item.prices = item.prices ?? [];
	item.creator_name = item.creator_name ?? "";
	return item;
}

export async function getMaterials(query: MaterialPageRequest): Promise<MaterialPage> {
	const raw = await request({ url: "/material", method: "GET", data: parseToObject(query) });
	const result = parse<MaterialPagePayload>(raw);
	if (result == null) throw new Error("物料列表响应为空，请重试");
	const list = (result.list ?? []).map((item: Material): Material => normalizeMaterial(item));
	return { total: result.total ?? 0, list } as MaterialPage;
}

// 与 Web MaterialModelRemoteSelect 一致：模糊发现编号，去重后由用户选择完整编号。
export async function findMaterialModels(keyword: string): Promise<string[]> {
	const query = keyword.trim();
	if (query == "") return [];
	const result = await getMaterials({ page: 1, size: 10, model: query });
	const models: string[] = [];
	result.list.forEach((item: Material) => {
		const model = item.model.trim();
		if (model != "" && !models.includes(model) && models.length < 10) models.push(model);
	});
	return models;
}

export async function getMaterial(id: string): Promise<Material> {
	const raw = await request({ url: "/material/info", method: "GET", data: { id } });
	const result = parse<Material>(raw);
	if (result == null || (result.id ?? "").length == 0) throw new Error("未找到物料资料");
	return normalizeMaterial(result);
}

export async function getMaterialCategories(): Promise<MaterialCategory[]> {
	const raw = await request({ url: "/material/category", method: "GET" });
	const result = parse<MaterialCategoryListPayload>({ items: raw });
	return result?.items ?? [];
}

export async function updateMaterial(form: MaterialRequest): Promise<void> {
	await request({ url: "/material", method: "PUT", data: parseToObject(form) });
}
