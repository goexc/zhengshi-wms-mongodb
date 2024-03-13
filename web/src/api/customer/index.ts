//统一管理客户相关接口

import request from "@/utils/request.ts";
import {
  CustomerRequest,
  CustomersRequest,
  CustomersResponse,
  CustomerStatusRequest, CustomerTransactionAddRequest,
  CustomerTransactionPageRequest, CustomerTransactionsResponse
} from "@/api/customer/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取客户列表接口
  CUSTOMER_LIST_URL = "/customer/list",
  
  //添加客户、修改客户、获取客户分页接口
  CUSTOMER_URL = '/customer',

  //修改客户状态/删除客户
  CUSTOMER_STATUS_URL = '/customer/status',
  
  //重新统计客户应收账款
  CUSTOMER_RECOUNT = '/customer/recount',

  //获取客户交易流水分页、添加客户交易记录
  CUSTOMER_TRANSACTION_URL = '/customer/transaction',
}

//获取客户列表接口
export const reqCustomerList = () =>
  request.get<any, CustomersResponse>(API.CUSTOMER_LIST_URL, { params: {} });


//获取客户分页接口
export const reqCustomers = (req:CustomersRequest) => {
  return request.get<any, CustomersResponse>(API.CUSTOMER_URL, {
    params: req,
  });
}

//添加与修改客户的接口方法
export const reqAddOrUpdateCustomer = (data: CustomerRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.CUSTOMER_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.CUSTOMER_URL, data);
  }
};

//修改客户状态[包括删除]
export const reqChangeCustomerStatus = (data: CustomerStatusRequest) =>
  request.patch<any, baseResponse>(API.CUSTOMER_STATUS_URL, data);

//重新统计客户应收账款
export const reqRecountReceivableBalance = ()=>
    request.get<any,baseResponse>(API.CUSTOMER_RECOUNT, {})

//获取客户交易流水分页接口
export const reqCustomerTransactions = (req:CustomerTransactionPageRequest) => {
  return request.get<any, CustomerTransactionsResponse>(API.CUSTOMER_TRANSACTION_URL, {
    params: req,
  });
}

//添加客户交易记录接口
export const reqAddCustomerTransaction = (data: CustomerTransactionAddRequest) => {
    return request.post<any, baseResponse>(API.CUSTOMER_TRANSACTION_URL, data);
};