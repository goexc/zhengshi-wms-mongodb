//大仓库
import { createPinia } from "pinia";
import piniaPluginPersistedstate from "pinia-plugin-persistedstate";

//创建大仓库
const pinia = createPinia();

//引入持久化插件
pinia.use(piniaPluginPersistedstate);

//对外暴露
export default pinia;
