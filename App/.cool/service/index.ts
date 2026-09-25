import { config } from "@/config";
import { user } from "../store/user";
export type RequestOptions = {
 url: string; method?: RequestMethod; data?: any; params?: any; header?: any; timeout?: number;
};
export type Response = { code?: number; message?: string; data?: any };
function responseMessage(value: UTSJSONObject, keys: string[]): string {
 for (let i = 0; i < keys.length; i++) {
  const item = value[keys[i]];
  if (typeof item == "string" && item.trim() != "") return item;
 }
 return "";
}
export function errorMessage(error: any | null, fallback: string): string {
 if (error == null) return fallback;
 if (typeof error == "string") return error.trim() != "" ? error : fallback;
 if (error instanceof Error) return error.message.trim() != "" ? error.message : fallback;
 if (typeof error != "object" || Array.isArray(error)) return fallback;
 const value = error as UTSJSONObject;
 const message = responseMessage(value, ["message", "msg", "errMsg"]);
 return message.length > 0 ? message : fallback;
}
function perform(options: RequestOptions, envelope: boolean): Promise<any | null> {
 const path = options.url;
 const publicRequest = path == "/auth/login";
 if (!publicRequest && !user.hasCredential()) {
  user.logout();
  return new Promise<any | null>((_resolve, reject) => {
   reject({ code: 401, message: "登录已过期，请重新登录" } as Response);
  });
 }
 const credential = user.token;
 const revision = user.revision.value;
 return new Promise((resolve, reject) => {
  uni.request({
   url: config.baseUrl + path, method: options.method ?? "GET",
   data: options.data ?? options.params ?? {}, timeout: options.timeout ?? 20000,
   header: { Authorization: publicRequest ? null : credential, ...((options.header ?? {}) as UTSJSONObject) },
   success(res) {
    // A late login failure/success must not clear or replace a newer session either.
    if (credential != user.token || revision != user.revision.value) {
     reject({ code: 401, message: "会话已变更，请重新查询" } as Response); return;
    }
    const raw = res.data;
    const body = raw != null && typeof raw == "object" && !Array.isArray(raw) ? raw as UTSJSONObject : null;
    const status = res.statusCode;
    const rawCode = body == null ? null : body["code"];
    const validCode = typeof rawCode == "number" && Number.isFinite(rawCode as number) && Math.floor(rawCode as number) == rawCode;
    const code = validCode ? rawCode as number : 0;
    const message = body == null ? "" : responseMessage(body, ["msg", "message"]);
    if (status >= 200 && status < 300 && code == 200 && body != null) {
     resolve(envelope ? body : body["data"] ?? null); return;
    }
    if (status >= 200 && status < 300 && code == 204 && path == "/account/menu") {
     const empty: UTSJSONObject = { menus: [] as UTSJSONObject[], buttons: [] as UTSJSONObject[] };
     resolve(envelope ? { code: 200, msg: message, data: empty } : empty); return;
    }
    if (!publicRequest && (status == 401 || code == 401)) user.logout();
    reject({ code: status >= 400 ? status : validCode ? code : 502,
     message: message != "" ? message : (status == 403 || code == 403 ? "没有访问权限" : !validCode && status >= 200 && status < 300 ? "服务响应格式异常，请重试" : "请求失败，请稍后重试") } as Response);
   },
   fail(err) { reject({ code: 0, message: err.errMsg } as Response); }
  });
 });
}
export function request(options: RequestOptions): Promise<any | null> { return perform(options, false); }
export function requestEnvelope(options: RequestOptions): Promise<any | null> { return perform(options, true); }
