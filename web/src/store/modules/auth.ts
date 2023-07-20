import { defineStore } from "pinia";
import {  getAuthMenuListApi } from "@/api/modules/login";
import { AuthState } from "@/store/interface";
import { getFlatMenuList } from "@/utils/menu.ts";
import piniaPersistConfig from "@/config/piniaPersist.ts";

export const useAuthStore = defineStore({
  id: "zs-auth",
  state: (): AuthState => ({
    // 按钮权限列表
    // authButtonList: {},
    authButtonList: [],
    // 菜单权限列表
    authMenuList: [],
    // 当前页面的 router name，用来做按钮权限筛选
    routeName: "",
  }),
  getters: {
    // 按钮权限列表
    authButtonListGet: (state) => state.authButtonList,
    // 菜单权限列表 ==> 这里的菜单没有经过任何处理
    authMenuListGet: (state) => state.authMenuList,
    // 菜单权限列表 ==> 扁平化之后的一维数组菜单，主要用来添加动态路由
    flatMenuListGet: (state) => getFlatMenuList(state.authMenuList),
  },
  actions: {
    // Get AuthButtonList
    // async getAuthButtonList() {
    //   const { data } = await getAuthButtonListApi();
    //   this.authButtonList = data;
    // },
    // Get AuthMenuList
    async getAuthMenuList() {
      const { data } = await getAuthMenuListApi();
      this.authMenuList = data.menus.sort((a, b) => a.sort_id - b.sort_id);
      this.authButtonList = data.buttons;
    },
    // Set RouteName
    async setRouteName(name: string) {
      this.routeName = name;
    },
  },
  // persist: piniaPersistConfig('zs-auth')
});
