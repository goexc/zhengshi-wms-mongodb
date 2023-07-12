//统一管理用户相关接口

//统一管理接口
import request from "@/utils/request";
import {
  loginForm,
  loginResponse,
  accountInfoResponse,
  accountMenusResponse,
} from "./types";

enum API {
  LOGIN_URL = "/auth/login",
  LOGOUT_URL = "/auth/logout",
  ACCOUNT_INFO_URL = "/account/profile",
  ACCOUNT_MENUS_URL = "/account/menu",
}

//暴露请求函数
//登录接口方法
export const reqLogin = (data: loginForm) =>
  request.post<any, loginResponse>(API.LOGIN_URL, data);
//获取用户信息接口方法
export const reqAccountInfo = () =>
  request.get<any, accountInfoResponse>(API.ACCOUNT_INFO_URL);

//退出登录
export const reqLogout = () => request.post<any, any>(API.LOGOUT_URL);

//获取用户的菜单id列表
export const reqAccountMenus = () =>
  request.get<any, accountMenusResponse>(API.ACCOUNT_MENUS_URL);
