//统一管理客户相关接口

import request from "@/utils/request.ts";
import {CustomerRequest, CustomersRequest, CustomersResponse, CustomerStatusRequest} from "@/api/customer/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取客户列表接口
  SUPPLIER_LIST_URL = "/customer/list",
  
  //添加客户、修改客户、获取客户分页接口
  SUPPLIER_URL = '/customer',

  //修改客户状态/删除客户
  SUPPLIER_STATUS_URL = '/customer/status',
}

//获取客户列表接口
export const reqCustomerList = () =>
  request.get<any, CustomersResponse>(API.SUPPLIER_LIST_URL, { params: {} });


//获取客户分页接口
export const reqCustomers = (req:CustomersRequest) => {
  return request.get<any, CustomersResponse>(API.SUPPLIER_URL, {
    params: req,
  });
}

//添加与修改客户的接口方法
export const reqAddOrUpdateCustomer = (data: CustomerRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.SUPPLIER_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.SUPPLIER_URL, data);
  }
};

//修改客户状态[包括删除]
export const reqChangeCustomerStatus = (data: CustomerStatusRequest) =>
  request.patch<any, baseResponse>(API.SUPPLIER_STATUS_URL, data);
