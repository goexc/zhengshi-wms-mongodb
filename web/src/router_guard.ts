//通过vue-router插件实现模板路由配置
import {RouteLocationNormalized} from "vue-router";
//进度条
import * as nProgress from "nprogress";
import "nprogress/nprogress.css";
import useUserStore from "@/store/module/account.ts";
import router from "@/router";
import setting from "@/setting.ts";
import {ElMessage} from "element-plus";


//页面刷新，变量初始化
let pageLoad = 0;

//进度条配置
nProgress.configure({showSpinner: false});

//路由鉴权
//全局前置守卫
router.beforeEach(
  async (to: RouteLocationNormalized, _from: RouteLocationNormalized, next) => {
    //修改标签页title
    document.title = (setting.title + " - " + to.meta.title) as string;
    //在钩子函数外，pinia 还没有被挂载，
    // 但是在前置守卫函数中，pinia 已经被挂载了，
    // 所以获取全局状态，需要在钩子函数中进行
    const userStore = useUserStore();

    //初始化进度条
    nProgress.start();

    console.log('to.matched:', to.matched)

    //用户未登录：可以访问login
    //用户登录：不能访问login[指向首页]，可以访问其余路由
    const token = userStore.token;
    //判断用户登录状态
    if (token) {
      //登录成功，不能访问login
      if (to.path === "/login") {
        return next({path: "/"});
      } else {
        //登录成功，访问其余路由
        //由用户信息，放行
        //没有用户信息，发起请求
        if (!userStore.name) {
          try {
            //请求用户信息
            console.log('请求用户信息')
            await userStore.info();

            //放行
            //next({ ...to}) 能保证找不到路由的时候重新执行beforeEach钩子
            //next({ replace: true}) 保证刷新时不允许用户后退
            return next({...to, replace: true}); //防止不存在的路由不跳转404
          } catch (e: any | unknown) {
            ElMessage.error(e);
            //情形1：token过期
            //情形2：token被篡改
            //请求退出登录
            await userStore.logout();
            return next({path: "/login", query: {redirect: to.path}});
          }
        } else {
          //有用户信息

          // console.log("pageLoad:", pageLoad);
          if (pageLoad === 0) {
            pageLoad++;
            //页面刷新，重新请求路由列表
            console.log('页面刷新，重新请求路由列表')
            await userStore.info()
            return next({...to, replace: true});
          } else {
            return next();
          }
        }
      }
    } else {
      if (to.path === "/login") {
        return next();
      } else {
        return next({path: "/login", query: {redirect: to.path}});
      }
    }

  }
);

//全局后置守卫
router.afterEach(
  (_to: RouteLocationNormalized, _from: RouteLocationNormalized) => {
    nProgress.done();
  }
);