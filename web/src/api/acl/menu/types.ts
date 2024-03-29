//菜单列表
export interface MenuListResponse {
  code: number;
  msg: string;
  data: Menu[];
}

export interface Perms {
  menus: Menu[],
  buttons: Button[],
}

export interface Menu {
  id: string;
  parent_id: string; //上级菜单id
  type: number; //菜单类型：1.菜单，2.按钮
  sort_id: number; //排序
  path: string; //路由路径
  name: string; //菜单名称
  component: string; //路由组件
  // icon: string; //    元信息：图标
  // transition: string; //    元信息：过渡动画
  // hidden: boolean; //    元信息：是否隐藏
  // fixed: boolean; //    元信息：是否固定
  // is_full: boolean; //    元信息：是否全屏
  // perms: string; //    权限标识
  meta: MetaProps;
  remark: string; //备注
  created_at?: number;
  updated_at?: number;
  children?: Menu[];
}

export interface MenuOptions {
  id: string;
  path: string;
  name: string;
  parent_id: string; //上级菜单id
  type: number; //菜单类型：1.菜单，2.按钮
  component?: string | (() => Promise<unknown>);
  redirect?: string;
  sort_id: number;
  meta: MetaProps;
  children?: MenuOptions[];
}

interface MetaProps {
  icon: string;
  title: string;
  activeMenu?: string;
  isLink?: string;
  transition: string; //特效
  hidden: boolean;
  is_full: boolean;
  fixed: boolean;
  perms?: string; //    权限标识
  isKeepAlive: boolean;
}

export interface Button {
  name: string;//按钮名称
  icon: string; //按钮图标
  perms: string; //按钮权限
}

//添加、修改菜单
export interface MenuRequest {
  id: string;
  parent_id: string; //上级菜单id
  type: number; //菜单类型：1.菜单，2.按钮
  sort_id: number; //排序
  name: string;
  path: string; //路由路径
  component: string; //路由组件
  icon: string; //    元信息：图标
  transition: string; //    元信息：过渡动画
  hidden: boolean; //    元信息：是否隐藏
  fixed: boolean; //    元信息：是否固定
  perms: string; //    权限标识
  remark: string;
}

//菜单状态
export interface MenuRemoveRequest {
  id: string;
}

//分配菜单
export interface MenuRequest {
  id: string;
  menus_id: string[]; //菜单id
}
