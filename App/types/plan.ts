export type Plan = {
	id: string;
	type: string;
	status: string;
	customer_id: string;
	customer_name: string;
	supplier_id: string;
	supplier_name: string;
	material_id: string;
	material_name: string;
	material_model: string;
	material_image: string;
	material_unit: string;
	material_quantity: number;
	deadline?: number;
};

export type PlanPage = { total: number; list: Plan[] | null };
