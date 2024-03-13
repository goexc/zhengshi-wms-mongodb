//统一管理出库单相关接口

import request from "@/utils/request.ts";
import {
  OutboundOrderCheckRequest,
  OutboundOrderConfirmRequest, OutboundOrderDepartureRequest, OutboundOrderMaterialRevise,
  OutboundOrderMaterialsRequest,
  OutboundOrderMaterialsResponse, OutboundOrderPackRequest, OutboundOrderPickRequest,
  OutboundOrderReceiptRequest,
  OutboundOrderRequest,
  OutboundOrdersRequest,
  OutboundOrdersResponse, OutboundOrderSummaryRequest, OutboundOrderSummaryResponse,
  OutboundOrderWeighRequest
} from "@/api/outbound/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //出库单分页接口(不携带物料列表)
  OUTBOUND_PAGE_URL = '/outbound/page',
  //出库单分页接口(携带物料列表)
  OUTBOUND_PAGE2_URL = '/outbound/page2',
  //添加出库单、修改出库单、删除出库单
  OUTBOUND_URL = '/outbound',
  //物料列表
  OUTBOUND_MATERIALS_URL = '/outbound/materials',
  //确认出库单
  OUTBOUND_CONFIRM_URL='/outbound/confirm',
  //确认拣货
  OUTBOUND_PICK_URL='/outbound/pick',
  //确认打包
  OUTBOUND_PACK_URL='/outbound/pack',
  //确认称重
  OUTBOUND_WEIGH_URL='/outbound/weigh',
  //确认出库
  OUTBOUND_DEPARTURE_URL='/outbound/departure',
  //签收
  OUTBOUND_RECEIPT_URL='/outbound/receipt',

  //审核出库单
  OUTBOUND_CHECK_URL = '/outbound/check',

  //修改出库单物料单价
  OUTBOUND_REVISE_URL = '/outbound/revise',

  //出库汇总
  OUTBOUND_SUMMARY_URL = '/outbound/summary',


}


//获取出库单分页接口(不携带物料列表)
export const reqOutboundOrders = (req: OutboundOrdersRequest) => {
  return request.get<any, OutboundOrdersResponse>(API.OUTBOUND_PAGE_URL, {
    params: req,
  });
}

//获取出库单分页接口(携带物料列表)
export const reqOutboundOrders2 = (req: OutboundOrdersRequest) => {
  return request.get<any, OutboundOrdersResponse>(API.OUTBOUND_PAGE2_URL, {
    params: req,
  });
}

//获取出库单物料列表
export const reqOutboundOrderMaterials = (req: OutboundOrderMaterialsRequest) => {
  return request.get<any, OutboundOrderMaterialsResponse>(API.OUTBOUND_MATERIALS_URL, {
    params: req,
  });
}

//添加出库单的接口方法
export const reqAddOutboundOrder = (data: OutboundOrderRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.OUTBOUND_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.OUTBOUND_URL, data);
  }
};

//删除出库单
export const reqRemoveOutboundOrder = (id: string) =>
  request.delete<any, baseResponse>(API.OUTBOUND_URL, {params: {id: id}});

//确认出库单
export const reqConfirmOutboundOrder = (req: OutboundOrderConfirmRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_CONFIRM_URL, req);

//确认拣货
export const reqPickOutboundOrder = (req: OutboundOrderPickRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_PICK_URL,  req);

//确认打包
export const reqPackOutboundOrder = (req:OutboundOrderPackRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_PACK_URL,  req);

//确认称重，更新物料重量
export const reqWeighOutboundOrder = (req: OutboundOrderWeighRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_WEIGH_URL,  req);

//确认出库
export const reqDepartureOutboundOrder = (req: OutboundOrderDepartureRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_DEPARTURE_URL, req);


//签收
export const reqReceiptOutboundOrder = (req: OutboundOrderReceiptRequest) =>
// export const reqReceiptOutboundOrder = (code:string) =>
  request.patch<any, baseResponse>(API.OUTBOUND_RECEIPT_URL, req);

//审核出库单
export const reqCheckOutboundOrder = (data: OutboundOrderCheckRequest) =>
  request.patch<any, baseResponse>(API.OUTBOUND_CHECK_URL, data)

//修改出库单物料单价
export const reqReviseOutboundOrder = (data: OutboundOrderMaterialRevise) =>
    request.patch<any, baseResponse>(API.OUTBOUND_REVISE_URL, data)

//出库汇总
export const reqOutboundOrderSummary = (req: OutboundOrderSummaryRequest) => {
  return request.get<any, OutboundOrderSummaryResponse>(API.OUTBOUND_SUMMARY_URL, {
    params: req,
  });
}
