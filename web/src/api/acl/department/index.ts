//部门相关api

//部门管理模块接口地址
import request from "@/utils/request.ts";
import {
  DepartmentListResponse,
  DepartmentRequest,
  DepartmentRemoveRequest,
} from "@/api/acl/department/types.ts";
import { baseResponse } from "@/api/types.ts";

enum API {
  //添加部门、修改部门、删除部门、获取部门列表接口
  DEPARTMENT_URL = "/department",
}

//获取部门列表的接口方法
export const reqDepartmentList = () =>
  request.get<any, DepartmentListResponse>(API.DEPARTMENT_URL);

//添加或修改部门的接口方法
export const reqAddOrUpdateDepartment = (data: DepartmentRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.DEPARTMENT_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.DEPARTMENT_URL, data);
  }
};

//删除部门
export const reqRemoveDepartment = (data: DepartmentRemoveRequest) =>
  request.delete<any, baseResponse>(API.DEPARTMENT_URL, {
    params: data,
  });
