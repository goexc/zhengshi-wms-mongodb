//图片管理模块接口地址
import request from "@/utils/request.ts";
import {baseResponse} from "@/api/types.ts";
import {ImageRequest, ImagesRequest, ImagesResponse} from "@/api/image/types.ts";

export enum API {
  //添加图片、修改图片、获取图片分页接口
  IMAGE_URL = "/images",
}

//获取图片分页接口
export const reqImages = (req:ImagesRequest) =>
  request.get<any, ImagesResponse>(API.IMAGE_URL, {
    params: req,
  });

//添加与修改图片的接口方法
export const reqAddImage = (data: ImageRequest) => {
  return request.post<any, baseResponse>(API.IMAGE_URL, data);
};
