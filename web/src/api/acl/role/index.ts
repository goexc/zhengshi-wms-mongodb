//角色相关api
import { baseResponse } from "@/api/types";
import request from "@/utils/request.ts";
import {
  RoleIdRequest,
  RoleListRequest,
  RoleListResponse,
  RoleMenusRequest,
  RoleMenusResponse,
  RoleRequest,
  RolesRequest,
  RolesResponse,
  RoleStatusRequest,
} from "./types";

//角色管理模块接口地址
enum API {
  //获取角色列表接口
  ROLE_LIST_URL = "/role/list",
  //添加角色、修改角色、获取角色分页接口
  ROLE_URL = "/role",
  //修改角色状态
  ROLE_STATUS_URL = "/role/status",
  //分配角色菜单、角色菜单列表
  ROLE_MENUS_URL = "/role/menus",
}

//获取角色列表的接口方法
export const reqRoleList = (req: RoleListRequest) =>
  request.get<any, RoleListResponse>(API.ROLE_LIST_URL, { params: req });

//获取角色分页的接口方法
export const reqRoles = (req: RolesRequest) =>
  request.get<any, RolesResponse>(API.ROLE_URL, { params: req });

//添加或修改角色的接口方法
export const reqAddOrUpdateRole = (data: RoleRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ROLE_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.ROLE_URL, data);
  }
};

//修改角色状态[包括删除]
export const reqChangeRoleStatus = (data: RoleStatusRequest) =>
  request.patch<any, baseResponse>(API.ROLE_STATUS_URL, data);

//分配角色菜单
export const reqChangeRoleMenus = (data: RoleMenusRequest) =>
  request.post<any, baseResponse>(API.ROLE_MENUS_URL, data);

//查询角色菜单列表
export const reqRoleMenus = (req: RoleIdRequest) =>
  request.get<any, RoleMenusResponse>(API.ROLE_MENUS_URL, { params: req });
