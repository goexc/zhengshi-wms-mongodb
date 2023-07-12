//菜单相关api
import request from "@/utils/request";
import { baseResponse } from "@/api/types";
import { MenuListResponse, MenuRequest, MenuStatusRequest } from "./types";

//菜单管理模块接口地址
enum API {
  //获取菜单列表接口
  MENU_LIST_URL = "/menu/list",
  //添加菜单
  ADD_MENU_URL = "/menu",
  //修改菜单
  UPDATE_MENU_URL = "/menu",
  //修改菜单状态
  MENU_STATUS_URL = "/menu/status",
}

//获取菜单列表的接口方法
export const reqMenuList = () =>
  request.get<any, MenuListResponse>(API.MENU_LIST_URL);

//添加或修改菜单的接口方法
export const reqAddOrUpdateMenu = (data: MenuRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ADD_MENU_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.UPDATE_MENU_URL, data);
  }
};

//修改菜单状态[包括删除]
export const reqChangeMenuStatus = (data: MenuStatusRequest) =>
  request.patch<any, baseResponse>(API.MENU_STATUS_URL, data);
