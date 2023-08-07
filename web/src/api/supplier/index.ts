//统一管理供应商相关接口

import request from "@/utils/request.ts";
import {SupplierRequest, SuppliersRequest, SuppliersResponse, SupplierStatusRequest} from "@/api/supplier/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取供应商列表接口
  SUPPLIER_LIST_URL = "/supplier/list",
  
  //添加供应商、修改供应商、获取供应商分页接口
  SUPPLIER_URL = '/supplier',

  //修改供应商状态/删除供应商
  SUPPLIER_STATUS_URL = '/supplier/status',
}

//获取供应商列表接口
export const reqSupplierList = () =>
  request.get<any, SuppliersResponse>(API.SUPPLIER_LIST_URL, { params: {} });


//获取供应商分页接口
export const reqSuppliers = (req:SuppliersRequest) => {
  return request.get<any, SuppliersResponse>(API.SUPPLIER_URL, {
    params: req,
  });
}

//添加与修改供应商的接口方法
export const reqAddOrUpdateSupplier = (data: SupplierRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.SUPPLIER_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.SUPPLIER_URL, data);
  }
};

//修改供应商状态[包括删除]
export const reqChangeSupplierStatus = (data: SupplierStatusRequest) =>
  request.patch<any, baseResponse>(API.SUPPLIER_STATUS_URL, data);
