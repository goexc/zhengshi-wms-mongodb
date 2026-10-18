export type DocumentHistory = { action: string; remark: string; time: number; operator_name: string };
export type CustomerDocument = {
 id: string; customer_id: string; customer_name: string; bill_number: string; invoice_number: string;
 amount: number; received_date: number; issue_date: number; due_date: number; redemption_date: number;
 invoice_date: number; tax_rate: number; amount_excluding_tax: number; tax_amount: number; total_amount: number;
 status: string; annex: string[]; redemption_annex: string[]; remark: string; version: number;
 transaction_id: string; creator_name: string; created_at: number; updated_at: number; history: DocumentHistory[];
};
export type DocumentSummary = { count: number; amount: number; pending_amount: number; overdue_amount: number; redeemed_amount: number; voided_amount: number; amount_excluding_tax: number; tax_amount: number; total_amount: number };
export type CustomerDocumentPage = { total: number; list: CustomerDocument[]; summary: DocumentSummary };
export function emptyDocumentSummary(): DocumentSummary { return { count: 0, amount: 0, pending_amount: 0, overdue_amount: 0, redeemed_amount: 0, voided_amount: 0, amount_excluding_tax: 0, tax_amount: 0, total_amount: 0 }; }
export function emptyCustomerDocument(): CustomerDocument { return { id: "", customer_id: "", customer_name: "", bill_number: "", invoice_number: "", amount: 0, received_date: 0, issue_date: 0, due_date: 0, redemption_date: 0, invoice_date: 0, tax_rate: 0, amount_excluding_tax: 0, tax_amount: 0, total_amount: 0, status: "", annex: [], redemption_annex: [], remark: "", version: 0, transaction_id: "", creator_name: "", created_at: 0, updated_at: 0, history: [] }; }
