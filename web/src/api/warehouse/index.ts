
//仓库管理模块接口地址
import request from "@/utils/request.ts";
import {
  WarehouseRequest,
  WarehousesRequest,
  WarehousesResponse,
  WarehouseStatusRequest, WarehouseTreeResponse
} from "@/api/warehouse/types.ts";
import {baseResponse} from "@/api/types.ts";

enum API {
  //获取库区列表接口
  WAREHOUSE_LIST_URL = "/warehouse/list",
  //添加仓库、修改仓库、获取仓库分页接口
  WAREHOUSE_URL = "/warehouse",
  //修改仓库状态
  WAREHOUSE_STATUS_URL = "/warehouse/status",
  //仓库树
  WAREHOUSE_TREE_URL = "/warehouse/tree",
}

//获取仓库列表接口
export const reqWarehouseTree = () =>
  request.get<any, WarehouseTreeResponse>(API.WAREHOUSE_TREE_URL, { params: {} });



//获取仓库列表接口
export const reqWarehouseList = () =>
  request.get<any, WarehousesResponse>(API.WAREHOUSE_LIST_URL, { params: {} });

//获取仓库分页接口
export const reqWarehouses = (req:WarehousesRequest) => {
  return request.get<any, WarehousesResponse>(API.WAREHOUSE_URL, {
    params: req,
  });
}
//添加与修改仓库的接口方法
export const reqAddOrUpdateWarehouse = (data: WarehouseRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.WAREHOUSE_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.WAREHOUSE_URL, data);
  }
};

//修改仓库状态[包括删除]
export const reqChangeWarehouseStatus = (data: WarehouseStatusRequest) =>
  request.patch<any, baseResponse>(API.WAREHOUSE_STATUS_URL, data);
