//库区相关api
import request from "@/utils/request.ts";
import {
  ZoneRequest, ZonesRequest, ZonesResponse, ZoneStatusRequest,
} from "@/api/warehouse_zone/types.ts";
import {baseResponse} from "@/api/types.ts";

//库区管理模块接口地址
enum API {
  //添加库区、修改库区、获取库区分页接口
  ZONE_URL = "/warehouse_zone",
  //修改库区状态
  ZONE_STATUS_URL = "/warehouse_zone/status",
}

//获取库区分页接口
export const reqZones = (req: ZonesRequest) =>
  request.get<any, ZonesResponse>(API.ZONE_URL, {
    params: req,
  });

//添加与修改仓库的接口方法
export const reqAddOrUpdateZone = (data: ZoneRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.ZONE_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.ZONE_URL, data);
  }
};

//修改仓库状态[包括删除]
export const reqChangeZoneStatus = (data: ZoneStatusRequest) =>
  request.patch<any, baseResponse>(API.ZONE_STATUS_URL, data);
