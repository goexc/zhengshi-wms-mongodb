//系统设置
import { defineStore } from "pinia";
import piniaPersistConfig from "@/config/piniaPersist";

const useSettingStore = defineStore("Setting", {
  state: () => {
    return {
      theme: "", //默认为空。dark:夜间模式
      color: "", //自定义主题颜色
    };
  },
  actions: {
    //切换主题
    changeTheme(night: boolean) {
      //获取html根节点
      const html = document.documentElement;

      this.theme = night ? "dark" : "";
      html.className = this.theme;
    },
    //更改主题颜色
    changeColor(color: string) {
      //获取html根节点
      const html = document.documentElement;

      //通知js修改根节点的样式
      this.color = color;
      html.style.setProperty("--el-color-primary", color);
    },
    //初始化设置
    initSetting() {
      console.log("初始化设置");
      //获取html根节点
      const html = document.documentElement;
      html.className = this.theme;
      html.style.setProperty("--el-color-primary", this.color);
    },
  },
  getters: {},
  persist: piniaPersistConfig("p_setting"),
});

export default useSettingStore;
