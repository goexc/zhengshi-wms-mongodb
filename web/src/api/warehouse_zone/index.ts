//库区相关api
import request from "@/utils/request";
import {
  WarehouseZoneListRequest,
  WarehouseZoneListResponse,
} from "@/api/warehouse_zone/types.ts";

//库区管理模块接口地址
enum API {
  //获取库区列表接口
  ZONE_LIST_URL = "/warehouse_zone/list",
  //获取库区分页接口
  ZONES_URL = "/warehouse_zone",
  //添加库区
  ADD_ZONE_URL = "/warehouse_zone",
  //修改库区
  UPDATE_ZONE_URL = "/warehouse_zone",
  //修改库区状态
  ZONE_STATUS_URL = "/warehouse_zone/status",
}

//获取库区列表接口
export const reqWarehouseZoneList = (req: WarehouseZoneListRequest) =>
  request.get<any, WarehouseZoneListResponse>(API.ZONE_LIST_URL, {
    params: req,
  });
