//layout组件相关配置仓库
import { defineStore } from "pinia";

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
  persist: {
    enabled: true,
    strategies: [
      {
        key: "p_layout",
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        // paths: ['collapse']
      },
    ],
  },
});

export default useLayoutSettingStore;
