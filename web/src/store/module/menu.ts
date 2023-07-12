//创建菜单相关的小仓库
import { defineStore } from "pinia";
import { Menu, MenuListResponse } from "@/api/acl/menu/types.ts";
import { reqMenuList } from "@/api/acl/menu";

const useMenuStore = defineStore("Menu", {
  state: () => {
    return {
      list: <Menu[]>[],
    };
  },
  //异步|逻辑
  //actions 内部不能写箭头函数
  actions: {
    async getMenus() {
      const res: MenuListResponse = await reqMenuList();
      if (res.code === 200) {
        this.list = res.data;
      }
    },
  },
  getters: {},
  //持久化
  persist: {
    enabled: true,
    strategies: [
      {
        key: "p_menu",
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        // paths: ['token']
      },
    ],
  },
});

export default useMenuStore;
