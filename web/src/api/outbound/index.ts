//统一管理出库单相关接口

import request from "@/utils/request.ts";
import {
  OutboundReceiptCheckRequest, OutboundReceiptMaterialRequest,
  OutboundReceiptRequest,
  OutboundReceiptsRequest,
  OutboundReceiptsResponse
} from "@/api/outbound/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取出库单列表接口
  OUTBOUND_LIST_URL = "/outbound/receipt/list",
  
  //添加出库单、修改出库单、获取出库单分页接口、删除出库单
  OUTBOUND_URL = '/outbound/receipt',

  //审核出库单
  OUTBOUND_CHECK_URL='/outbound/receipt/check',

  //修改物料信息
  OUTBOUND_MATERIAL_URL='/outbound/receipt/material'
}

//获取出库单列表接口
export const reqOutboundReceiptList = () =>
  request.get<any, OutboundReceiptsResponse>(API.OUTBOUND_LIST_URL, { params: {} });


//获取出库单分页接口
export const reqOutboundReceipts = (req:OutboundReceiptsRequest) => {
  return request.get<any, OutboundReceiptsResponse>(API.OUTBOUND_URL, {
    params: req,
  });
}

//添加与修改出库单的接口方法
export const reqAddOrUpdateOutboundReceipt = (data: OutboundReceiptRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.OUTBOUND_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.OUTBOUND_URL, data);
  }
};

//删除出库单
export const reqRemoveOutboundReceipt = (id:string) =>
  request.delete<any, baseResponse>(API.OUTBOUND_URL, {params: {id:id}});

//审核出库单
export const reqCheckOutboundReceipt = (data:OutboundReceiptCheckRequest)=>
    request.patch<any, baseResponse>(API.OUTBOUND_CHECK_URL, data)

//修改物料信息
export const reqUpdateOutboundReceiptMaterial = (data:OutboundReceiptMaterialRequest) =>
    request.patch<any, baseResponse>(API.OUTBOUND_MATERIAL_URL,data)