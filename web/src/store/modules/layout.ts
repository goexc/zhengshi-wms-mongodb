//layout组件相关配置仓库
import { defineStore } from "pinia";
import piniaPersistConfig from "@/config/piniaPersist.ts";

const useLayoutSettingStore = defineStore("Layout", {
  state: () => {
    return {
      collapse: false, //菜单是否折叠
      refresh: false, //控制刷新效果
    };
  },
  actions: {},
  getters: {},
  //持久化
  persist: piniaPersistConfig("zs-layout"),
});

export default useLayoutSettingStore;
