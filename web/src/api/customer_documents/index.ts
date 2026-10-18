import request from '@/utils/request.ts';
import type { CustomerDocumentKind, CustomerDocument, DocumentQuery, DocumentPage, DocumentSaveRequest, RedemptionRequest, DocumentVoidRequest, PaymentOption, DocumentResponse } from './types';

const endpoint = (kind: CustomerDocumentKind) => `/customer/${kind}`;

export const getDocuments = (kind: CustomerDocumentKind, query: DocumentQuery) =>
  request.get<unknown, DocumentResponse<DocumentPage>>(endpoint(kind), { params: query });
export const getDocument = (kind: CustomerDocumentKind, id: string) =>
  request.get<unknown, DocumentResponse<CustomerDocument>>(`${endpoint(kind)}/detail`, { params: { id } });
export const saveDocument = (kind: CustomerDocumentKind, data: DocumentSaveRequest) => data.id
  ? request.put<unknown, DocumentResponse<CustomerDocument>>(endpoint(kind), data)
  : request.post<unknown, DocumentResponse<CustomerDocument>>(endpoint(kind), data);
export const redeemAcceptance = (data: RedemptionRequest) =>
  request.post<unknown, DocumentResponse<CustomerDocument>>('/customer/acceptance/redeem', data);
export const reverseAcceptance = (data: DocumentVoidRequest) =>
  request.post<unknown, DocumentResponse<CustomerDocument>>('/customer/acceptance/reverse', data);
export const voidDocument = (kind: CustomerDocumentKind, data: DocumentVoidRequest) =>
  request.post<unknown, DocumentResponse<CustomerDocument>>(`${endpoint(kind)}/void`, data);
export const getAcceptancePayments = (query: { customer_id: string; amount: number; page: number; size: number }) =>
  request.get<unknown, DocumentResponse<{ total: number; list: PaymentOption[] }>>('/customer/acceptance/payments', { params: query });
