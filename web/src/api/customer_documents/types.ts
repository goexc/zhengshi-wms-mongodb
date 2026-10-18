export type CustomerDocumentKind = 'acceptance' | 'invoice';

export interface DocumentHistory {
  action: string;
  remark: string;
  time: number;
  operator: string;
  operator_name: string;
}

export interface CustomerDocument {
  id: string;
  customer_id: string;
  customer_name: string;
  status: string;
  annex: string[];
  remark: string;
  version: number;
  created_at: number;
  updated_at: number;
  creator_name: string;
  history: DocumentHistory[];
  bill_number?: string;
  amount?: number;
  received_date?: number;
  issue_date?: number;
  due_date?: number;
  redemption_date?: number;
  redemption_annex?: string[];
  transaction_id?: string;
  invoice_number?: string;
  invoice_date?: number;
  tax_rate?: number;
  amount_excluding_tax?: number;
  tax_amount?: number;
  total_amount?: number;
}

export interface DocumentQuery {
  page: number;
  size: number;
  customer_id?: string;
  keyword?: string;
  status?: string;
  date_start?: number;
  date_end?: number;
  due_start?: number;
  due_end?: number;
  redemption_start?: number;
  redemption_end?: number;
  due_state?: string;
  tax_rate?: string;
}

export interface DocumentSummary {
  count: number;
  amount?: number;
  pending_amount?: number;
  redeemed_amount?: number;
  overdue_amount?: number;
  voided_amount?: number;
  amount_excluding_tax?: number;
  tax_amount?: number;
  total_amount?: number;
}

export interface DocumentPage {
  total: number;
  list: CustomerDocument[];
  summary: DocumentSummary;
}

export interface DocumentSaveRequest {
  id?: string;
  version?: number;
  idempotency_key?: string;
  customer_id: string;
  annex: string[];
  remark: string;
  bill_number?: string;
  amount?: number;
  received_date?: number;
  issue_date?: number;
  due_date?: number;
  invoice_number?: string;
  invoice_date?: number;
  tax_rate?: number;
  amount_excluding_tax?: number;
  tax_amount?: number;
  total_amount?: number;
}

export interface RedemptionRequest {
  id: string;
  version: number;
  redemption_date: number;
  redemption_annex: string[];
  remark: string;
  idempotency_key: string;
  transaction_id?: string;
}

export interface DocumentVoidRequest {
  id: string;
  version: number;
  reason: string;
  idempotency_key: string;
}

export interface PaymentOption {
  id: string;
  time: number;
  amount: number;
  remark: string;
  code: string;
  customer_id: string;
  annex: string[];
}

export interface DocumentResponse<T> { code: number; msg: string; data: T }
