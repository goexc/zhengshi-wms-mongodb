export type Partner = {
	id: string;
	type: string;
	code: string;
	name: string;
	image: string;
	legal_representative: string;
	unified_social_credit_identifier: string;
	address: string;
	contact: string;
	manager: string;
	email: string;
	remark: string;
	status: string;
	receivable_balance?: number | null;
	credit_balance?: number | null;
	level?: number | null;
	create_by?: string;
	created_at?: number;
	updated_at?: number;
};

export type PartnerPage = { total: number; list: Partner[] | null };

export type PartnerQuery = {
	page: number;
	size: number;
	name: string;
	code: string;
	manager: string;
	contact: string;
	email: string;
	level?: number;
};

export type CustomerTransaction = {
	type: string;
	transaction_type: string;
	direction: string;
	status: string;
	source_type: string;
	source_code: string;
	time: number;
	amount?: number | null;
	remark: string;
	annex: string;
};

export type CustomerTransactionPage = { total: number; list: CustomerTransaction[] | null };
