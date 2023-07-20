//用户相关api
import { baseResponse } from "@/api/types";
import request from "@/utils/request.ts";
import {
  UserRequest,
  UserRolesRequest,
  UsersRequest,
  UsersResponse,
  UserStatusRequest,
} from "@/api/acl/user/types.ts";

//用户管理模块接口地址
enum API {
  //添加用户、修改用户、获取用户分页接口
  USER_URL = "/user",
  //修改用户状态
  USER_STATUS_URL = "/user/status",
  //修改用户角色
  USER_ROLES_URL = "/user/roles",
}

//获取用户分页的接口方法
export const reqUsers = (req: UsersRequest) =>
  request.get<any, UsersResponse>(API.USER_URL, { params: req });

//添加或修改用户的接口方法
export const reqAddOrUpdateUser = (data: UserRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.USER_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.USER_URL, data);
  }
};

//修改用户状态[包括删除]
export const reqChangeUserStatus = (data: UserStatusRequest) =>
  request.patch<any, baseResponse>(API.USER_STATUS_URL, data);

//修改用户角色
export const reqChangeUserRoles = (data: UserRolesRequest) =>
  request.patch<any, baseResponse>(API.USER_ROLES_URL, data);
