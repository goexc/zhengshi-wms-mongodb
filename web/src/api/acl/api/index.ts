//api相关api
import request from "@/utils/request.ts";
import { baseResponse } from "@/api/types";
import {Api, ApiListResponse} from "./types";

//api管理模块接口地址
enum API {
  //添加api、修改api、删除api、获取api列表接口
  API_URL = "/api",
}

//获取api列表的接口方法
export const reqApiList = () =>
  request.get<any, ApiListResponse>(API.API_URL);

//添加或修改api的接口方法
export const reqAddOrUpdateApi = (data: Api) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.API_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.API_URL, data);
  }
};

//删除api
export const reqDeleteApi = (data: {id:string}) =>
  request.delete<any, baseResponse>(API.API_URL, {params: data});
