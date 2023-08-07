//统一管理承运商相关接口

import request from "@/utils/request.ts";
import {CarrierRequest, CarriersRequest, CarriersResponse, CarrierStatusRequest} from "@/api/carrier/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取承运商列表接口
  CARRIER_LIST_URL = "/carrier/list",
  
  //添加承运商、修改承运商、获取承运商分页接口
  CARRIER_URL = '/carrier',

  //修改承运商状态/删除承运商
  CARRIER_STATUS_URL = '/carrier/status',
}

//获取承运商列表接口
export const reqCarrierList = () =>
  request.get<any, CarriersResponse>(API.CARRIER_LIST_URL, { params: {} });


//获取承运商分页接口
export const reqCarriers = (req:CarriersRequest) => {
  return request.get<any, CarriersResponse>(API.CARRIER_URL, {
    params: req,
  });
}

//添加与修改承运商的接口方法
export const reqAddOrUpdateCarrier = (data: CarrierRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.CARRIER_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.CARRIER_URL, data);
  }
};

//修改承运商状态[包括删除]
export const reqChangeCarrierStatus = (data: CarrierStatusRequest) =>
  request.patch<any, baseResponse>(API.CARRIER_STATUS_URL, data);
