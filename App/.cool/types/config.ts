export type WxConfig = { debug: boolean };
export type Config = {
 name: string; version: string; locale?: string; website: string; host: string;
 baseUrl: string; showDarkButton: boolean; isCustomTabBar: boolean; backTop: boolean; wx: WxConfig;
};
export type PluginConfig = { options?: UTSJSONObject; install(app: VueApp): UTSJSONObject | void };
