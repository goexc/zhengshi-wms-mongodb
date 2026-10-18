import { defineStore } from "pinia";
import {  getAuthMenuListApi } from "@/api/modules/login";
import { AuthState } from "@/store/interface";
import { prepareNavigationMenus, getFlatMenuList } from "@/utils/menu.ts";
import {Button, Menu} from "@/api/acl/menu/types.ts";
// import piniaPersistConfig from "@/config/piniaPersist.ts";

export const useAuthStore = defineStore("zs-auth", {
  state: (): AuthState => ({
    // 按钮权限列表
    // authButtonList: {},
    authButtonList: <Button[]>[],
    // 菜单权限列表
    authMenuList: <Menu[]>[],
    // 当前页面的 router name，用来做按钮权限筛选
    routeName: "",
  }),
  getters: {
    // 按钮权限列表
    authButtonListGet: (state) => state.authButtonList,
    // 菜单权限列表 ==> 已移除退役页面入口
    authMenuListGet: (state) => state.authMenuList,
    // 菜单权限列表 ==> 扁平化之后的一维数组菜单，主要用来添加动态路由
    flatMenuListGet: (state) => getFlatMenuList(state.authMenuList),
  },
  actions: {
    // Get AuthMenuList
    async getAuthMenuList() {
      const { data } = await getAuthMenuListApi();
      this.authMenuList = prepareNavigationMenus(data.menus);
      this.authButtonList = data.buttons;
    },
    // Set RouteName
    async setRouteName(name: string) {
      this.routeName = name;
    },
  },
  // persist: piniaPersistConfig('zs-auth')
});
