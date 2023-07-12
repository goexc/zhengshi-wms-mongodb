//创建用户相关的小仓库
import {defineStore} from "pinia";
import {reqAccountInfo, reqLogin, reqLogout} from "@/api/user";
import {
  accountInfoResponse,
  button,
  loginForm,
  loginResponse,
} from "@/api/user/types";
//引入常量路由
import {anyRoute, asyncRoute, constantRoute} from "@/router/routes";
import {baseResponse} from "@/api/types";
import {RouteRecordRaw} from "vue-router";
import router from "@/router";
//引入深拷贝方法
import {cloneDeep} from "lodash";
import {Menu} from "@/api/acl/menu/types.ts";

//***********Vite 的组件懒加载***************
//获取文件夹及其嵌套的多级子文件夹的 .vue 文件夹，注意 ** 是关键。
//一个 *：匹配当前目录下的文件；
//两个 *：匹配当前目录及其嵌套的全部子目录下的文件；
const views =  await import.meta.glob("@/views/**/*.vue")
const layout =  await import.meta.glob("@/layout/**/*.vue")
// const viteComponent = await import.meta.glob("@/views/**/*.vue")

const generateRoutes = async (menus: Menu[]) => {
  let routes: Array<RouteRecordRaw> = []

  // console.log('views:', views)
  // console.log('layout:', layout)

  routes = menus.map((r) => {

    let route: RouteRecordRaw = <RouteRecordRaw>{
      path: r.path,
      name: r.name,
      meta: {
        title: r.name,
        type: 1,
        hidden: r.hidden, //在菜单中是否隐藏
        fixed: r.fixed, //在菜单中是否固定
        transition: r.transition,
        icon: r.icon,
        isFull: r.is_full, //是否全屏
      },
      // component: viteComponent["/src/views" + r.component + ".vue"],
    }

    if(!!views["/src/views" + r.component + ".vue"]){
      route.component = views["/src/views" + r.component + ".vue"]
    }else{
      // route.component = layout["/src" + r.component + ".vue"]
    }

    // console.log('初始化路由', route.path, ',', route.component)

    if (!!r.children && r.children.length > 0) {
      generateRoutes(r.children).then(res => {
        route.children = res
      })
    }

    return route
  })

  return routes
}

//创建用户小仓库
const useUserStore = defineStore("User", {
  //小仓库存储的数据
  state: () => {
    return {
      token: "", //用户唯一标识token
      // menuRoutes: constantRoute,
      menuRoutes: <RouteRecordRaw[]>[],
      name: "", //用户名
      avatar: "", //头像
      buttons: <button[]>[], //按钮权限
    };
  },
  //异步|逻辑
  //actions 内部不能写箭头函数
  actions: {
    //登录
    async login(loginForm: loginForm) {
      const res: loginResponse = await reqLogin(loginForm);
      if (res.code === 200) {
        this.token = res.data.token as string;
        this.menuRoutes = [...constantRoute];
        return "ok";
      }
      return Promise.reject(new Error(res.msg));
    },
    //获取用户信息
    async info() {
      const res: accountInfoResponse = await reqAccountInfo();
      if (res.code === 200) {
        //存储用户信息
        this.name = res.data.name;
        this.avatar = res.data.avatar;
        //按钮权限数组
        this.buttons = res.data.buttons;
        //动态路由
        let asyncRoutes = await generateRoutes(res.data.routes);
        // console.log('路由数组：', asyncRoutes)
        await this.loadAsyncRoutes(asyncRoutes);

        return "ok";
      }

      return Promise.reject(new Error(res.msg));
    },
    //退出登录
    async logout() {
      //1.向服务器发出退出登录的请求
      const res: baseResponse = await reqLogout();

      //2.清空仓库中的相关数据[token|name|avatar]
      if (res.code === 200) {
        this.token = "";
        this.name = "";
        this.avatar = "";
        this.menuRoutes = [...constantRoute];
        this.buttons = [];
        return "ok";
      } else {
        return Promise.reject(new Error(res.msg));
      }
    },
    //加载动态路由
    async loadAsyncRoutes(asyncRoutes) {
      this.menuRoutes = [...asyncRoutes, ...anyRoute];
      asyncRoutes.map((route: RouteRecordRaw) => {
        // if(!!views[route.component]){
        //   console.log('view组件：', route.component)
        //   route.component = views[route.component]
        // }else{
        //   console.log('layout组件：', route.component)
        //   route.component = layout[route.component]
        // }


        // console.log('组件'+route.path+':', route.component)
        if (route.meta.isFull) { //全屏路由
          router.addRoute(route)
        } else {
          router.addRoute("layout", route)
        }
      })
    },
  },
  //
  getters: {},
  //持久化
  persist: {
    enabled: true,
    strategies: [
      {
        key: "p_user",
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        // paths: ['token']
      },
    ],
  },
});

export default useUserStore;
