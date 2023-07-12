import { createApp } from "vue";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import zh from "element-plus/dist/locale/zh-cn.mjs";
//暗黑模式css变量
import 'element-plus/theme-chalk/dark/css-vars.css';
//svg插件需要配置代码
import "virtual:svg-icons-register";
//引入自定义插件对象：注册整个项目全局组件
import globalComponent from "@/components";
//引入模板的全局样式
import "@/styles/index.scss";

import App from "./App.vue";
import router from "./router";
import pinia from "./store";
import * as ElementPlusIconsVue from "@element-plus/icons-vue";
// font css
import "@/assets/fonts/font.scss";

const app = createApp(App);

app.use(ElementPlus, {
  locale: zh,
  size: "large",
});

//安装自定义插件：注册全局组件
app.use(globalComponent);

//安装仓库
app.use(pinia);

//注册模板路由
app.use(router);
//引入路由守卫
import "./router_guard";

//注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component);
}

app.mount("#app");
