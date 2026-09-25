export type OutboundQuery = {
	page: number; size: number; code: string; customer_id: string; model: string;
	status: string; type: string; is_pack: number; is_weigh: number;
};

export type OutboundOrder = {
	id: string; code: string; status: string; type: string;
	customer_id: string; customer_name: string; supplier_id: string; supplier_name: string;
	carrier_name: string; carrier_cost: number | null; other_cost: number | null;
	total_amount: number | null; departure_time: number; receipt_time: number;
	remark: string;
};

export type OutboundPage = { total: number; list: OutboundOrder[] };

export type OutboundMaterial = {
	id: string; order_code: string; material_id: string; index: number;
	name: string; model: string; specification: string; unit: string;
	price: number | null; quantity: number | null; weight: number | null;
	departure_date: number; receipt_date: number;
};

export type OutboundUnitTotal = { unit: string; quantity: number };
export type OutboundTotals = {
	orderCount: number; lineCount: number; amount: number | null;
	units: OutboundUnitTotal[]; hasMissingQuantity: boolean;
};

export type CustomerChoice = { id: string; name: string; code: string };
