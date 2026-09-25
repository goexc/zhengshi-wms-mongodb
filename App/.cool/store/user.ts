import { ref, computed } from "vue";
import { parseToObject, storage, keys } from "../utils";
import { router } from "../router";
import { request, errorMessage } from "../service";
import type { UserInfo } from "../types/user";
export type Token = { token: string; exp: number };
export type LoginData = {
 name: string; avatar: string; mobile: string; email: string;
 department_id: string; department_name: string; token: string; exp: number;
};
const tokenKey = "wms.auth.token";
const userKey = "wms.auth.user";
const schemaKey = "wms.auth.schema";
function profileText(value: UTSJSONObject, key: string): string {
 const field = value[key];
 return typeof field == "string" ? field : "";
}
export class User {
 info = ref<UserInfo | null>(null);
 token: string | null = null;
 permissions = ref<string[]>([]);
 menuPaths = ref<string[]>([]);
 permissionsReady = ref(false);
 permissionError = ref("");
 loading = ref(false);
 revision = ref(0);
 private profileRequest = 0;
 private permissionRequest = 0;
 constructor() {
  if (storage.get(schemaKey) != "1") {
   const oldKeys = ["auth", "auth_token", "userInfo", "token", "refresh_token", "refreshToken", "cy_auth_storage_schema", "cy_app_update_force_block", "cy_app_update_pending_install"];
   oldKeys.forEach((key) => storage.remove(key));
   storage.set(schemaKey, "1", 0);
  }
  if (!storage.isExpired(tokenKey)) {
   const token = storage.get(tokenKey);
   if (typeof token == "string" && token.length > 0) {
    this.token = token;
    const cached = storage.get(userKey);
    if (cached != null && typeof cached == "object" && !Array.isArray(cached)) {
     try { this.set(cached); } catch (_) { storage.remove(userKey); }
    }
   }
  }
 }
 hasCredential(): boolean {
  if (this.token == null || this.token == "") return false;
  if (storage.isExpired(tokenKey)) { this.clear(); return false; }
  return true;
 }
 setToken(value: Token) {
  if (typeof value.token != "string" || value.token.trim() == "" || typeof value.exp != "number" || !Number.isFinite(value.exp) || Math.floor(value.exp) != value.exp) throw new Error("登录凭证无效或已过期");
  const remaining = value.exp - Math.floor(Date.now() / 1000);
  if (value.token == "" || remaining <= 0) throw new Error("登录凭证无效或已过期");
  this.token = value.token;
  this.revision.value++;
  storage.set(tokenKey, value.token, remaining);
 }
 setInfo(info: UserInfo) {
  this.info.value = info;
  storage.set(userKey, parseToObject(info), 0);
 }
 set(data: any) {
 if (data == null || typeof data != "object" || Array.isArray(data)) throw new Error("个人资料响应格式异常");
 const obj = data as UTSJSONObject;
 if (typeof obj["name"] != "string") throw new Error("个人资料响应缺少账号名称");
 this.setInfo({name: profileText(obj, "name"), avatar: profileText(obj, "avatar"), mobile: profileText(obj, "mobile"), email: profileText(obj, "email"), department_id: profileText(obj, "department_id"), department_name: profileText(obj, "department_name")});
 }
 async get() {
  if (!this.hasCredential() || this.loading.value) return;
  this.loading.value = true;
  const token = this.token;
  const revision = this.revision.value;
  const ticket = ++this.profileRequest;
  try {
   try {
    const data = await request({ url: "/account/profile" });
    if (token == this.token && revision == this.revision.value && ticket == this.profileRequest && data != null) this.set(data);
   } catch (_) { /* A transient profile failure keeps the valid cached identity. */ }
   if (token != this.token || revision != this.revision.value || ticket != this.profileRequest) return;
   await this.loadPermissions();
  } finally {
   if (token == this.token && revision == this.revision.value && ticket == this.profileRequest) this.loading.value = false;
  }
 }
 async loadPermissions() {
  if (!this.hasCredential()) return;
  const token = this.token;
  const revision = this.revision.value;
  const ticket = ++this.permissionRequest;
  this.permissionError.value = "";
  try {
   const raw = await request({ url: "/account/menu" });
   if (token != this.token || revision != this.revision.value || ticket != this.permissionRequest) return;
   if (raw == null || typeof raw != "object" || Array.isArray(raw)) throw new Error("权限响应格式异常");
   const data = raw as UTSJSONObject;
   const fields = keys(data);
   if (!fields.includes("menus") || !fields.includes("buttons")) throw new Error("权限响应不完整");
   const perms: string[] = [];
   const paths: string[] = [];
   function walk(items: any | null) {
    if (items == null) return;
    if (!Array.isArray(items)) throw new Error("权限菜单响应格式异常");
    (items as any[]).forEach((rawItem: any) => {
     if (rawItem == null || typeof rawItem != "object" || Array.isArray(rawItem)) throw new Error("权限菜单响应格式异常");
     const item = rawItem as UTSJSONObject;
     const path = item["path"];
     if (typeof path == "string" && !paths.includes(path as string)) paths.push(path as string);
     const rawMeta = item["meta"];
     if (rawMeta != null && (typeof rawMeta != "object" || Array.isArray(rawMeta))) throw new Error("权限菜单信息格式异常");
     const meta = (rawMeta ?? {}) as UTSJSONObject;
     const code = profileText(meta, "perms").trim();
     if (code != "" && !perms.includes(code)) perms.push(code);
     walk(item["children"]);
    });
   }
   walk(data["menus"]);
   const buttons = data["buttons"];
   if (buttons != null && !Array.isArray(buttons)) throw new Error("操作权限响应格式异常");
   ((buttons ?? ([] as any[])) as any[]).forEach((rawButton: any) => {
    if (rawButton == null || typeof rawButton != "object" || Array.isArray(rawButton)) throw new Error("操作权限响应格式异常");
    const code = profileText(rawButton as UTSJSONObject, "perms").trim();
    if (code != "" && !perms.includes(code)) perms.push(code);
   });
   this.permissions.value = perms;
   this.menuPaths.value = paths;
   this.permissionsReady.value = true;
   uni.$emit("wms.permissions.ready");
  } catch (err) {
   if (token != this.token || revision != this.revision.value || ticket != this.permissionRequest) return;
   this.permissions.value = [];
   this.menuPaths.value = [];
   this.permissionsReady.value = false;
   this.permissionError.value = errorMessage(err, "权限加载失败，请重试");
  }
 }
 hasPermission(code: string): boolean { return this.permissionsReady.value && this.permissions.value.includes(code); }
 canAccess(feature: string): boolean {
  if (!this.permissionsReady.value) return false;
  const prefix = feature == "material" ? "material:material:" : feature == "outbound" ? "outbound:order:" : feature == "customer" ? "business_partner:customer:" : feature == "supplier" ? "business_partner:supplier:" : "plan:";
  if (this.permissions.value.some((value) => value.startsWith(prefix))) return true;
  const needle = feature == "customer" ? "customer" : feature == "supplier" ? "supplier" : feature;
  return this.menuPaths.value.some((path) => path.split("/").includes(needle));
 }
 isNull(): boolean { return this.info.value == null; }
 clear() {
  storage.remove(tokenKey); storage.remove(userKey); storage.remove("router-params");
  this.token = null; this.info.value = null; this.permissions.value = []; this.menuPaths.value = [];
  this.permissionsReady.value = false; this.permissionError.value = ""; this.loading.value = false;
  this.profileRequest++; this.permissionRequest++;
  this.revision.value++;
  uni.$emit("wms.session.changed");
 }
 logout() { this.clear(); router.login(); }
}
export const user = new User();
export const userInfo = computed<UserInfo | null>(() => user.info.value);
