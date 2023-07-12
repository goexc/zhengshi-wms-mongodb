//部门相关api


//部门管理模块接口地址
import request from "@/utils/request.ts";
import {DepartmentListResponse, DepartmentRequest, DepartmentRemoveRequest} from "@/api/acl/department/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取部门列表接口
  DEPARTMENT_LIST_URL = "/department",
  //添加部门
  ADD_DEPARTMENT_URL = "/department",
  //修改部门
  UPDATE_DEPARTMENT_URL = "/department",
  //删除部门
  DEPARTMENT_STATUS_URL = "/department",
}

//获取部门列表的接口方法
export const reqDepartmentList = () =>
  request.get<any, DepartmentListResponse>(API.DEPARTMENT_LIST_URL);

//添加或修改部门的接口方法
export const reqAddOrUpdateDepartment = (data: DepartmentRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ADD_DEPARTMENT_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.UPDATE_DEPARTMENT_URL, data);
  }
};

//删除部门
export const reqRemoveDepartment = (data: DepartmentRemoveRequest) =>
  request.delete<any, baseResponse>(API.DEPARTMENT_STATUS_URL, {params: data});