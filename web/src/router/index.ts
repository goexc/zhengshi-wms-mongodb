import {createRouter, createWebHistory} from "vue-router";
import {useUserStore} from "@/store/modules/user.ts";
import {useAuthStore} from "@/store/modules/auth.ts";
import {LOGIN_URL, ROUTER_WHITE_LIST} from "@/config";
import NProgress from "@/config/nprogress";
import {initDynamicRouter} from "@/router/modules/dynamicRouter.ts";
import {errorRouter, staticRouter} from "@/router/modules/staticRouter.ts";
import {getNavigationTitle} from "@/utils/menu.ts";

/**
 * @description 📚 路由参数配置简介
 * @param path ==> 路由菜单访问路径
 * @param name ==> 路由 name (对应页面组件 name, 可用作 KeepAlive 缓存标识 && 按钮权限筛选)
 * @param redirect ==> 路由重定向地址
 * @param component ==> 视图文件路径
 * @param meta ==> 路由菜单元信息
 * @param meta.icon ==> 菜单和面包屑对应的图标
 * @param meta.title ==> 路由标题 (用作 document.title || 菜单的名称)
 * @param meta.activeMenu ==> 当前路由为详情页时，需要高亮的菜单
 * @param meta.isLink ==> 路由外链时填写的访问地址
 * @param meta.hidden ==> 是否在菜单中隐藏 (通常列表详情页需要隐藏)
 * @param meta.is_full ==> 菜单是否全屏 (示例：数据大屏页面)
 * @param meta.fixed ==> 菜单是否固定在标签页中 (首页通常是固定项)
 * @param meta.isKeepAlive ==> 当前路由是否缓存
 * */
const router = createRouter({
  history: createWebHistory(),
  routes: [...staticRouter, ...errorRouter],
  strict: false,
  scrollBehavior: () => ({left: 0, top: 0}),
});

/**
 * @description 路由拦截 beforeEach
 * */
router.beforeEach(async (to, from) => {
  const userStore = useUserStore();
  const authStore = useAuthStore();

  // 1.NProgress 开始
  NProgress.start();

  // 2.动态设置标题
  const title = import.meta.env.VITE_TITLE;
  const pageTitle = getNavigationTitle(to);
  document.title = pageTitle ? `${String(pageTitle)} - ${title}` : title;

  // 3.判断是访问登录页，有 Token 就在当前页面，没有 Token 重置路由到登录页
  if (to.path.toLocaleLowerCase() === LOGIN_URL) {
    if (userStore.token) return from.fullPath;
    resetRouter();
    return;
  }

  // 4.判断访问页面是否在路由白名单地址(静态路由)中，如果存在直接放行
  if (ROUTER_WHITE_LIST.includes(to.path)) return;

  // 5.判断是否有 Token，没有重定向到 login 页面
  if (!userStore.token) return {path: LOGIN_URL, replace: true};

  // 6.如果没有菜单列表，就重新请求菜单列表并添加动态路由
  if (!authStore.authMenuListGet.length) {
    await initDynamicRouter();
    return {...to, replace: true};
  }

  // 7.存储 routerName 做按钮权限筛选
  authStore.setRouteName(to.name as string);

  // 8.正常访问页面
  return;
});

/**
 * @description 重置路由
 * */
export const resetRouter = () => {
  const authStore = useAuthStore();
  authStore.flatMenuListGet.forEach((route) => {
    const {name} = route;
    if (name && router.hasRoute(name)) router.removeRoute(name);
  });
};

/**
 * @description 路由跳转错误
 * */
router.onError((error) => {
  NProgress.done();
  console.warn("路由错误", error.message);
});

/**
 * @description 路由跳转结束
 * */
router.afterEach(() => {
  NProgress.done();
});

export default router;
