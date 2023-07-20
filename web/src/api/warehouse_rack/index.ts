//货架相关api
import request from "@/utils/request.ts";
import {
  WarehouseRackListRequest,
  WarehouseRackListResponse,
} from "@/api/warehouse_rack/types.ts";

//货架管理模块接口地址
enum API {
  //获取货架列表接口
  RACK_LIST_URL = "/warehouse_rack/list",
  //添加货架、修改货架、获取货架分页接口
  RACK_URL = "/warehouse_rack",
  //修改货架状态
  RACK_STATUS_URL = "/warehouse_rack/status",
}

//获取货架列表接口
export const reqWarehouseRackList = (req: WarehouseRackListRequest) =>
  request.get<any, WarehouseRackListResponse>(API.RACK_LIST_URL, {
    params: req,
  });
