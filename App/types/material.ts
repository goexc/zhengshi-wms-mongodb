export type MaterialPrice = {
	price: number;
	since: number;
	customer_id: string;
	customer_name: string;
	source_type: string;
	source_quote_id: string;
	source_delivery_id: string;
	source_valid: boolean;
	source_invalid_reason: string;
};

export type Material = {
	id: string;
	image: string;
	category_id: string;
	category_name: string;
	name: string;
	material: string;
	specification: string;
	model: string;
	surface_treatment: string;
	strength_grade: string;
	quantity: number;
	unit: string;
	remark: string;
	prices: MaterialPrice[] | null;
	creator: string;
	creator_name: string;
	created_at: number;
	updated_at: number;
};

export type MaterialPage = { total: number; list: Material[] };
export type MaterialPagePayload = { total: number; list: Material[] | null };
export type MaterialPageRequest = { page: number; size: number; model: string };

export type MaterialRequest = {
	id: string;
	category_id: string;
	name: string;
	model: string;
	image: string;
	material: string;
	specification: string;
	surface_treatment: string;
	strength_grade: string;
	quantity: number;
	unit: string;
	remark: string;
};

export type MaterialCategory = {
	id: string;
	parent_id: string;
	name: string;
	status: string;
	children: MaterialCategory[] | null;
};

export type MaterialCategoryListPayload = { items: MaterialCategory[] | null };
