//统一管理物料相关接口
import {baseResponse} from "@/api/types.ts";
import request from "@/utils/request.ts";
import {
  MaterialCategoryIdRequest,
  MaterialCategoryRequest,
  MaterialCategorysResponse,
  MaterialDeliveryRebuildTaskPageResponse,
  MaterialDeliveryRebuildTaskRequest,
  MaterialDeliveryRebuildTaskResponse,
  MaterialIdRequest,
  MaterialPricesResponse,
  MaterialQuoteIdRequest,
  MaterialQuotePageRequest,
  MaterialQuotePageResponse,
  MaterialQuotePriceRequest,
  MaterialQuoteResponse,
  MaterialQuoteSaveRequest,
  MaterialRequest,
  MaterialsRequest,
  MaterialsResponse,
  NewCustomerMaterialExportRequest,
  NewCustomerMaterialRequest,
  NewCustomerMaterialResponse
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

  //客户新增物料
  MATERIAL_NEW_DELIVERY_URL = '/material/new_delivery',

  //物料报价
  MATERIAL_QUOTE_URL = '/material/quote',

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
export const reqMaterialPrices = (material_id:string, customer_id:string) =>
  request.get<any, MaterialPricesResponse>(API.MATERIAL_PRICE_URL, {params: {material_id:material_id,customer_id:customer_id}})

//删除物料单价
export const reqRemoveMaterialPrice = (id:string, customer_id:string, price:number) =>
  request.delete<any, baseResponse>(API.MATERIAL_PRICE_URL, {params: {id:id, customer_id:customer_id, price:price}})

//查询客户新增物料
export const reqNewCustomerMaterials = (req: NewCustomerMaterialRequest) =>
  request.get<any, NewCustomerMaterialResponse>(API.MATERIAL_NEW_DELIVERY_URL, {params: req});

//重建客户新增物料记录
export const reqExportNewCustomerMaterialQuotes = (data: NewCustomerMaterialExportRequest) =>
  request.post<any, Blob>(`${API.MATERIAL_NEW_DELIVERY_URL}/quote/export`, data, {responseType: 'blob'});

export const reqRebuildNewCustomerMaterials = () =>
  request.post<any, MaterialDeliveryRebuildTaskResponse>(`${API.MATERIAL_NEW_DELIVERY_URL}/rebuild`, {});

//查询最近一次客户新增物料重建任务
export const reqLatestMaterialDeliveryRebuildTask = () =>
  request.get<any, MaterialDeliveryRebuildTaskResponse>(`${API.MATERIAL_NEW_DELIVERY_URL}/rebuild/latest`);

//查询客户新增物料重建任务列表
export const reqMaterialDeliveryRebuildTasks = (req: MaterialDeliveryRebuildTaskRequest) =>
  request.get<any, MaterialDeliveryRebuildTaskPageResponse>(`${API.MATERIAL_NEW_DELIVERY_URL}/rebuild/tasks`, {params: req});

//保存物料报价
export const reqSaveMaterialQuote = (data: MaterialQuoteSaveRequest) => {
  if (data.id.trim().length === 0) {
    return request.post<any, MaterialQuoteResponse>(API.MATERIAL_QUOTE_URL, data);
  }
  return request.put<any, MaterialQuoteResponse>(API.MATERIAL_QUOTE_URL, data);
};

//查询报价单分页
export const reqMaterialQuotes = (req: MaterialQuotePageRequest) =>
  request.get<any, MaterialQuotePageResponse>(API.MATERIAL_QUOTE_URL, {params: req});

//查询报价单详情
export const reqMaterialQuoteInfo = (req: MaterialQuoteIdRequest) =>
  request.get<any, MaterialQuoteResponse>(`${API.MATERIAL_QUOTE_URL}/info`, {params: req});

//提交物料报价
export const reqSubmitMaterialQuote = (data: MaterialQuoteIdRequest) =>
  request.post<any, MaterialQuoteResponse>(`${API.MATERIAL_QUOTE_URL}/submit`, data);

//导出物料报价
export const reqExportMaterialQuote = (data: MaterialQuoteIdRequest) =>
  request.post<any, Blob>(`${API.MATERIAL_QUOTE_URL}/export`, data, {responseType: 'blob'});

//报价转最终定价
export const reqPriceMaterialQuote = (data: MaterialQuotePriceRequest) =>
  request.post<any, MaterialQuoteResponse>(`${API.MATERIAL_QUOTE_URL}/price`, data);

//作废物料报价
export const reqVoidMaterialQuote = (data: MaterialQuoteIdRequest) =>
  request.patch<any, baseResponse>(`${API.MATERIAL_QUOTE_URL}/void`, data);
