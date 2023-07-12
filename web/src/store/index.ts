//大仓库
import { createPinia } from "pinia";
import piniaPluginPersist from "pinia-plugin-persist";

//创建大仓库
const pinia = createPinia();

//引入持久化插件
pinia.use(piniaPluginPersist);

//对外暴露
export default pinia;
