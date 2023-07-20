//菜单相关api
import request from "@/utils/request.ts";
import { baseResponse } from "@/api/types";
import {Menu, MenuListResponse, MenuRemoveRequest, MenuStatusRequest} from "./types";

//菜单管理模块接口地址
enum API {
  //获取菜单列表接口
  MENU_LIST_URL = "/menu/list",
  //添加菜单、修改菜单
  ADD_MENU_URL = "/menu",
}

//获取菜单列表的接口方法
export const reqMenuList = () =>
  request.get<any, MenuListResponse>(API.MENU_LIST_URL);

//添加或修改菜单的接口方法
export const reqAddOrUpdateMenu = (data: Menu) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ADD_MENU_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.ADD_MENU_URL, data);
  }
};

//删除菜单
export const reqRemoveMenu = (data: MenuRemoveRequest) =>
  request.delete<any, baseResponse>(API.ADD_MENU_URL, {params:data});
