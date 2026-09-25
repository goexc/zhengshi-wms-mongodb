import type { PluginConfig } from "@/.cool/types/config";
import { router, useStore } from "@/.cool";
export default {
 install(_app: VueApp) {
  router.beforeEach((to, _from, next) => {
   const { user } = useStore();
   if (to.meta["isAuth"] == false || user.hasCredential()) { next(); return; }
   uni.setStorageSync("wms.returnTo", to.path);
   router.login();
  });
 }
} as PluginConfig;
