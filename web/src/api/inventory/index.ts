//统一管理库存管理相关接口

import {
  InventoryListRequest,
  InventoryListResponse,
  InventorysRequest,
  InventorysResponse
} from "@/api/inventory/types.ts";
import request from "@/utils/request.ts";

enum API {
  //库存管理分页接口
  INVENTORY_URL = "/inventory",
  //库存管理分页接口
  INVENTORY_RECORD_URL = "/inventory/record",
  //物料库存接口
  INVENTORY_LIST_URL = "/inventory/list",
}


//获取库存分页接口
export const reqInventory = (req:InventorysRequest) =>{
  return request.get<any, InventorysResponse>(API.INVENTORY_URL, { params: req });
}

//获取库存记录分页接口
export const reqInventoryRecord = (req:InventorysRequest) =>{
  return request.get<any, InventorysResponse>(API.INVENTORY_RECORD_URL, { params: req });
}

//获取物料库存接口(用于确认出库单时，查询物料库存)
export const reqInventoryList = (req:InventoryListRequest) => {
  return request.get<any, InventoryListResponse>(API.INVENTORY_LIST_URL, { params: req });
}