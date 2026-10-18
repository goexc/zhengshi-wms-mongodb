import { parse, request } from "@/.cool";
import type { CustomerDocument, CustomerDocumentPage } from "@/types/customer-documents";
import { emptyDocumentSummary } from "@/types/customer-documents";
import { materialImageUrl, previewMaterialImage } from "@/utils/material-image";
import { openWeb } from "@/uni_modules/cool-open-web";

export function documentKind(value: string): string { return value == "invoice" ? "invoice" : "acceptance"; }
export function documentTitle(kind: string): string { return kind == "invoice" ? "销售开票" : "客户承兑"; }
export function documentDate(value: number): string {
 if (!Number.isFinite(value) || value <= 0) return "—";
 const date = new Date((value + 8 * 3600) * 1000);
 const month = date.getUTCMonth() + 1; const day = date.getUTCDate();
 return `${date.getUTCFullYear()}-${month < 10 ? '0' : ''}${month}-${day < 10 ? '0' : ''}${day}`;
}
export function documentStatus(status: string, dueDate: number): string {
 if (status == "pending") return dueDate > 0 && Math.floor((Date.now() / 1000 + 8 * 3600) / 86400) > Math.floor((dueDate + 8 * 3600) / 86400) ? "待兑付 · 已到期" : "待兑付";
 if (status == "redeemed") return "已到账";
 if (status == "active") return "有效";
 if (status == "voided") return "已作废";
 return status;
}
export function documentHistoryAction(value: string): string {
 if (value == "create") return "新增";
 if (value == "update") return "修改";
 if (value == "redeem") return "兑付到账";
 if (value == "void") return "作废";
 if (value == "reverse") return "撤销到账";
 return value;
}
export async function fetchCustomerDocuments(kind: string, customerId: string, keyword: string, status: string, page: number): Promise<CustomerDocumentPage> {
 if (kind != "acceptance" && kind != "invoice") throw new Error("台账类型无效");
 if (customerId != "" && !/^[a-fA-F0-9]{24}$/.test(customerId)) throw new Error("客户信息无效，请返回客户列表重新选择");
 const data = await request({ url: `/customer/${kind}`, method: "GET", data: { customer_id: customerId, keyword: keyword.trim(), status, page, size: 20 } });
 const result = parse<CustomerDocumentPage>(data);
 if (result == null) throw new Error("台账响应为空，请重试");
 return { total: result.total ?? 0, list: result.list ?? [], summary: result.summary ?? emptyDocumentSummary() };
}
export async function fetchCustomerDocument(kind: string, id: string): Promise<CustomerDocument> {
 if (kind != "acceptance" && kind != "invoice") throw new Error("台账类型无效");
 if (!/^[a-fA-F0-9]{24}$/.test(id)) throw new Error("记录标识无效，请返回列表重新选择");
 const data = await request({ url: `/customer/${kind}/detail`, method: "GET", data: { id } });
 const result = parse<CustomerDocument>(data);
 if (result == null) throw new Error("详情响应为空，请重试");
 return result;
}
export function documentAttachmentLabel(reference: string): string {
 if (/\.(png|jpe?g|gif|webp|bmp)([?#].*)?$/i.test(reference.trim())) return "预览图片";
 if (/\.pdf([?#].*)?$/i.test(reference.trim())) return "打开 PDF";
 return "打开 / 下载文件";
}
export function openDocumentAttachment(reference: string): void {
 const url = materialImageUrl(reference);
 if (!url.startsWith("https://") && !url.startsWith("http://")) { uni.showToast({ title: "附件地址无效", icon: "none" }); return; }
 if (/\.(png|jpe?g|gif|webp|bmp)([?#].*)?$/i.test(reference.trim())) { previewMaterialImage(url); return; }
 if (!openWeb(url)) uni.showToast({ title: "无法打开，请检查系统浏览器", icon: "none" });
}
