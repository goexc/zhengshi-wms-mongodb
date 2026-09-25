export function formatMoney(value: number | null, digits: number = 3): string {
 if (value == null || !Number.isFinite(value)) return "—";
 return value.toFixed(digits);
}
export function formatQuantity(value: number): string {
 return Number.isFinite(value) ? `${parseFloat(value.toFixed(6))}` : "—";
}
export function formatDate(value: number): string {
 if (value <= 0 || !Number.isFinite(value)) return "—";
 const d = new Date(value * 1000);
 return `${d.getFullYear()}-${`${d.getMonth() + 1}`.padStart(2, "0")}-${`${d.getDate()}`.padStart(2, "0")}`;
}
export function companyName(value: string): string {
 return value.replace(/有限责任公司|有限公司/g, "");
}

// 沿用 Appx/name_encrypt 的去词顺序，仅用于客户价格中的名称展示。
export function materialPriceCustomerName(value: string | null): string {
 const name = (value ?? "").trim();
 if (name == "") return "未指定客户";
 const shortened = name
  .replace(/有限责任公司/g, "")
  .replace(/有限公司/g, "")
  .replace(/股份/g, "")
  .replace(/科技/g, "")
  .replace(/机械/g, "")
  .replace(/制造/g, "")
  .replace(/部件/g, "")
  .replace(/汽车/g, "")
  .replace(/农业/g, "")
  .replace(/装备/g, "")
  .replace(/工贸/g, "")
  .replace(/山东省/g, "")
  .replace(/山东/g, "")
  .replace(/潍坊市/g, "")
  .replace(/潍坊/g, "")
  .replace(/诸城市/g, "")
  .replace(/诸城/g, "")
  .trim();
 return shortened == "" ? name : shortened;
}
