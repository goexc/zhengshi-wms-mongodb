//统一管理入库单相关接口

import request from "@/utils/request.ts";
import {
  InboundReceiptCheckRequest, InboundReceiptMaterialRequest,
  InboundReceiptRequest,
  InboundReceiptsRequest,
  InboundReceiptsResponse
} from "@/api/inbound/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取入库单列表接口
  INBOUND_LIST_URL = "/inbound/receipt/list",
  
  //添加入库单、修改入库单、获取入库单分页接口、删除入库单
  INBOUND_URL = '/inbound/receipt',

  //修改入库单状态
  INBOUND_STATUS_URL = '/inbound/receipt/status',

  //审核入库单
  INBOUND_CHECK_URL='/inbound/receipt/check',

  //修改物料信息
  INBOUND_MATERIAL_URL='/inbound/receipt/material'
}

//获取入库单列表接口
export const reqInboundReceiptList = () =>
  request.get<any, InboundReceiptsResponse>(API.INBOUND_LIST_URL, { params: {} });


//获取入库单分页接口
export const reqInboundReceipts = (req:InboundReceiptsRequest) => {
  return request.get<any, InboundReceiptsResponse>(API.INBOUND_URL, {
    params: req,
  });
}

//添加与修改入库单的接口方法
export const reqAddOrUpdateInboundReceipt = (data: InboundReceiptRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.INBOUND_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.INBOUND_URL, data);
  }
};

//删除入库单
export const reqRemoveInboundReceipt = (id:string) =>
  request.delete<any, baseResponse>(API.INBOUND_URL, {params: {id:id}});

//审核入库单
export const reqCheckInboundReceipt = (data:InboundReceiptCheckRequest)=>
    request.patch<any, baseResponse>(API.INBOUND_CHECK_URL, data)

//修改物料信息
export const reqUpdateInboundReceiptMaterial = (data:InboundReceiptMaterialRequest) =>
    request.patch<any, baseResponse>(API.INBOUND_MATERIAL_URL,data)