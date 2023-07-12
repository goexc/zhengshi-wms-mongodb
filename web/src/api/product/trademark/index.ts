import request from "@/utils/request";
import { baseResponse } from "@/api/types";
import {
  WarehouseRequest,
  WarehousesResponse,
  WarehouseStatusRequest,
} from "@/api/product/trademark/types.ts";

//仓库管理模块接口地址
enum API {
  //获取库区列表接口
  WAREHOUSE_LIST_URL = "/warehouse/list",
  //获取仓库分页接口
  WAREHOUSES_URL = "/warehouse",
  //添加仓库
  ADD_WAREHOUSES_URL = "/warehouse",
  //修改仓库
  UPDATE_WAREHOUSES_URL = "/warehouse",
  //修改仓库状态
  WAREHOUSE_STATUS_URL = "/warehouse/status",
}

//获取仓库列表接口
export const reqTrademarkList = () =>
  request.get<any, WarehousesResponse>(API.WAREHOUSE_LIST_URL, { params: {} });

//获取仓库分页接口
export const reqTrademarks = (page: number, size: number) =>
  request.get<any, WarehousesResponse>(API.WAREHOUSES_URL, {
    params: { page: page, size: size },
  });

//添加与修改仓库的接口方法
export const reqAddOrUpdateTrademark = (data: WarehouseRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ADD_WAREHOUSES_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.UPDATE_WAREHOUSES_URL, data);
  }
};

//修改仓库状态[包括删除]
export const reqChangeTrademarkStatus = (data: WarehouseStatusRequest) =>
  request.patch<any, baseResponse>(API.WAREHOUSE_STATUS_URL, data);
