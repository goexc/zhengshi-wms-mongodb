import type { Config } from "@/.cool/types/config";
import { dev } from "./dev";
import { prod } from "./prod";
export const isDev = process.env.NODE_ENV == "development";
export const fileBaseUrl = "https://wms.file.goexc.cn/images/";
export const ignoreTokens: string[] = ["/auth/login"];
export const config = {
 name: "正时 WMS", version: "2.0.0", locale: "zh-cn", website: "",
 showDarkButton: false, isCustomTabBar: true, backTop: false, wx: { debug: false },
 ...(isDev ? dev() : prod())
} as Config;
