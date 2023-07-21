//货架相关api
import request from "@/utils/request.ts";
import {
  RackListRequest,
  RackListResponse,
  RackRequest,
  RacksRequest,
  RacksResponse,
  RackStatusRequest
} from "@/api/warehouse_rack/types.ts";
import {baseResponse} from "@/api/types.ts";

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
export const reqRackList = (req: RackListRequest) =>
  request.get<any, RackListResponse>(API.RACK_LIST_URL, {
    params: req,
  });

//获取货架分页接口
export const reqRacks = (req: RacksRequest) =>
  request.get<any, RacksResponse>(API.RACK_URL, {
    params: req,
  });

//添加与修改货架的接口方法
export const reqAddOrUpdateRack = (data: RackRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.RACK_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.RACK_URL, data);
  }
};

//修改货架状态[包括删除]
export const reqChangeRackStatus = (data: RackStatusRequest) =>
  request.patch<any, baseResponse>(API.RACK_STATUS_URL, data);
