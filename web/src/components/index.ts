//对外暴露插件对象

import SvgIcon from "@/components/SvgIcon/index.vue";
import Pagination from "@/components/Pagination/index.vue";
import Category from "@/components/Category/index.vue";
import PermsButton from "@/components/PermsButton/index.vue";
import PermsLink from "@/components/PermsLink/index.vue";
import { App } from "vue";
const allGlobalComponent: any = {
  SvgIcon,
  Pagination,
  Category,
  PermsButton,
  PermsLink,
};
export default {
  //install方法
  install(app: App) {
    Object.keys(allGlobalComponent).forEach((key) => {
      //注册为全局组件
      app.component(key, allGlobalComponent[key]);
    });
  },
};
