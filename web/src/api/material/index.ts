//统一管理物料相关接口
import {baseResponse} from "@/api/types.ts";
import request from "@/utils/request.ts";
import {
  MaterialCategoryIdRequest,
  MaterialCategoryRequest,
  MaterialCategorysResponse,
  MaterialIdRequest,
  MaterialRequest,
  MaterialsRequest,
  MaterialsResponse
} from "@/api/material/types.ts";

enum API {
  //添加物料、修改物料、删除物料、获取物料列表接口
  MATERIAL_URL = '/material',
  //获取物料分页接口
  MATERIAL_LIST_URL = "/material/list",


  //添加物料分类、修改物料分类、删除物料分类、获取物料分类列表接口
  MATERIAL_CATEGORY_URL = '/material/category',

  //查询/删除物料单价
  MATERIAL_PRICE_URL = '/material/price',

}

/*物料相关接口*/

//获取物料分页接口
export const reqMaterials = (req: MaterialsRequest) => {
  return request.get<any, MaterialsResponse>(API.MATERIAL_URL, {
    params: req,
  });
}

//添加与修改物料的接口方法
export const reqAddOrUpdateMaterial = (data: MaterialRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.MATERIAL_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.MATERIAL_URL, data);
  }
};

//删除物料
export const reqRemoveMaterial = (data: MaterialIdRequest) =>
  request.delete<any, baseResponse>(API.MATERIAL_URL, {params:data});


/*物料分类相关接口*/

//获取物料分类列表接口
export const reqMaterialCategoryList = () =>
  request.get<any, MaterialCategorysResponse>(API.MATERIAL_CATEGORY_URL, {params: {}});

//添加与修改物料分类的接口方法
export const reqAddOrUpdateMaterialCategory = (data: MaterialCategoryRequest) => {
  if (data.id.trim().length === 0) {
    //添加
    return request.post<any, baseResponse>(API.MATERIAL_CATEGORY_URL, data);
  } else {
    //修改
    return request.put<any, baseResponse>(API.MATERIAL_CATEGORY_URL, data);
  }
};

//删除物料分类的接口方法
export const reqRemoveMaterialCategory = (data: MaterialCategoryIdRequest) =>
  request.delete<any, baseResponse>(API.MATERIAL_CATEGORY_URL, {params: data});

//查询物料单价列表
export const reqMaterialPrices = (id:string) =>
  request.get<any, baseResponse>(API.MATERIAL_PRICE_URL, {params: {id:id}})

//删除物料单价列表
export const reqRemoveMaterialPrice = (id:string, price:number) =>
  request.delete<any, baseResponse>(API.MATERIAL_PRICE_URL, {params: {id:id, price:price}})