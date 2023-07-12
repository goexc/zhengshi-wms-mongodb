//对外暴露配置路由(常量路由)
import { RouteRecordRaw } from "vue-router";

//静态路由
export const constantRoute: RouteRecordRaw[] = [
  {
    //登录
    path: "/login",
    component: () => import("@/views/login/index.vue"),
    name: "login", //命名路由
    meta: {
      title: "登录", //菜单标题
      hidden: true, //在菜单中是否隐藏
      icon: "Promotion",
      type: 1,
      perms: "auth:login",
    },
  },
  {
    //登录成功后，展示数据
    path: "/",
    component: () => import("@/layout/index.vue"),
    name: "layout", //命名路由
    meta: {
      title: "layout", //菜单标题
      hidden: false, //在菜单中是否隐藏
      icon: "Avatar", //菜单文字左侧图标
      type: 1,
      perms: "home",
    },
    redirect: "/home",
    children: [
      {
        path: "/home",
        component: () => import("@/views/home/index.vue"),
        name: "home",
        meta: {
          title: "首页", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "HomeFilled",
          type: 1,
          perms: "home",
        },
      },
    ],
  },
  // {
  //   //大屏
  //   path: "/screen",
  //   component: () => import("@/views/screen/index.vue"),
  //   name: "screen", //命名路由
  //   meta: {
  //     title: "数据大屏", //菜单标题
  //     hidden: false, //在菜单中是否隐藏
  //     icon: "Platform",
  //     type: 1,
  //     perms: "screen",
  //   },
  // },

  {
    //404
    path: "/404",
    component: () => import("@/views/404/index.vue"),
    name: "404", //命名路由,
    meta: {
      title: "404", //菜单标题
      hidden: true, //在菜单中是否隐藏
      icon: "WarningFilled",
      type: 1,
      perms: "404",
    },
  },
];
/*

//异步路由
export const asyncRoute: RouteRecordRaw[] = [
  {
    //权限管理
    path: "/acl",
    component: () => import("@/layout/index.vue"),
    name: "Acl", //命名路由
    meta: {
      title: "权限管理", //菜单标题
      hidden: false, //在菜单中是否隐藏
      icon: "Lock", //菜单文字左侧图标
      type: 1,
      perms: "privilege",
    },
    redirect: "/acl/user",
    children: [
      {
        path: "/acl/user",
        component: () => import("@/views/acl/user/index.vue"),
        name: "user",
        meta: {
          title: "用户管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "User",
          type: 1,
          perms: "privilege:user",
        },
      },
      {
        path: "/acl/role",
        component: () => import("@/views/acl/role/index.vue"),
        name: "role",
        meta: {
          title: "角色管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "UserFilled",
          type: 1,
          perms: "privilege:role",
        },
      },
      {
        path: "/acl/menu",
        component: () => import("@/views/acl/menu/index.vue"),
        name: "menu",
        meta: {
          title: "菜单管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "Menu",
          type: 1,
          perms: "privilege:menu",
        },
      },
    ],
  },
  {
    path: "/product",
    component: () => import("@/layout/index.vue"),
    name: "product",
    meta: {
      title: "产品管理", //菜单标题
      hidden: false, //在菜单中是否隐藏
      icon: "Goods",
      type: 1,
      perms: "product",
    },
    redirect: "/product/trademark",
    children: [
      {
        path: "/product/trademark",
        component: () => import("@/views/product/trademark/index.vue"),
        name: "Trademark",
        meta: {
          title: "品牌管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "ShoppingCartFull",
          type: 1,
          perms: "product:trademark",
        },
      },
      {
        path: "/product/attr",
        component: () => import("@/views/product/attr/index.vue"),
        name: "Attr",
        meta: {
          title: "属性管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "Operation",
          type: 1,
          perms: "product:attr",
        },
      },
      {
        path: "/product/spu",
        component: () => import("@/views/product/spu/index.vue"),
        name: "Spu",
        meta: {
          title: "Spu管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "Box",
          type: 1,
          perms: "product:spu",
        },
      },
      {
        path: "/product/sku",
        component: () => import("@/views/product/sku/index.vue"),
        name: "Sku",
        meta: {
          title: "Sku管理", //菜单标题
          hidden: false, //在菜单中是否隐藏
          icon: "Notification",
          type: 1,
          perms: "product:sku",
        },
      },
    ],
  },
];
*/

//任意路由
export const anyRoute: RouteRecordRaw[] = [
  {
    path: "/:pathMatch(.*)*",
    redirect: "/404",
    name: "NotFound",
    meta: {
      title: "404", //菜单标题
      hidden: true, //在菜单中是否隐藏
      icon: "StarFilled",
      type: 1,
      perms: "any",
    },
  },
];
