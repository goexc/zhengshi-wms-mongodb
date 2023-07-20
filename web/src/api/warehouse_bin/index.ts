//货位相关api
import { baseResponse } from "@/api/types";
import request from "@/utils/request.ts";
import {
  WarehouseBinListRequest,
  WarehouseBinListResponse,
  WarehouseBinRequest,
  WarehouseBinsRequest,
  WarehouseBinsResponse,
  WarehouseBinStatusRequest,
} from "@/api/warehouse_bin/types.ts";

//货位管理模块接口地址
enum API {
  //获取已有货位接口
  BIN_LIST_URL = "/warehouse_bin/list",
  //添加货位、修改货位、获取货位分页接口
  BIN_URL = "/warehouse_bin",
  //修改货位状态
  BIN_STATUS_URL = "/warehouse_bin/status",
}

//获取货位列表的接口方法
export const reqWarehouseBinList = (req: WarehouseBinListRequest) =>
  request.get<any, WarehouseBinListResponse>(API.BIN_LIST_URL, { params: req });

//获取货位分页的接口方法
export const reqWarehouseBins = (req: WarehouseBinsRequest) =>
  request.get<any, WarehouseBinsResponse>(API.BIN_URL, { params: req });

//添加或修改货位的接口方法
export const reqAddOrUpdateWarehouseBin = (data: WarehouseBinRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.BIN_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.BIN_URL, data);
  }
};

//修改货位状态[包括删除]
export const reqChangeWarehouseBinStatus = (data: WarehouseBinStatusRequest) =>
  request.patch<any, baseResponse>(API.BIN_STATUS_URL, data);
