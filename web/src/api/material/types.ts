//物料分页请求
export interface MaterialsRequest {
  page: number; //页数
  size: number; //条数
  name: string; //物料名称
  image: string; //物料图片
  material: string; //材质：碳钢、不锈钢、合金钢等。
  specification: string; //规格：包括长度、宽度、厚度等尺寸信息。
  model: string; //型号：用于唯一标识和区分不同种类的钢材。
  surface_treatment: string; //表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
  strength_grade: string; //强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
}

//物料分页响应
export interface MaterialsResponse {
  code: number;
  msg: string;
  data: MaterialPaginate;
}

export interface MaterialPaginate {
  total: number;
  list: Material[];
}

export interface Material {
  id: string;
  name: string;//物料名称
  category_id: string; //物料分类id
  category_name: string; //物料分类名称
  image: string;//图片
  model: string;//型号：用于唯一标识和区分不同种类的钢材。
  material: string;//材质：碳钢、不锈钢、合金钢等。
  specification: string;//规格：包括长度、宽度、厚度等尺寸信息。
  surface_treatment: string;//表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
  strength_grade: string;//强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
  quantity: number;//安全库存
  unit: string;//计量单位，如个、箱、千克等
  remark: string;//备注
  prices: MaterialPrice[];//单价
  creator: string; //创建人id
  creator_name: string; //创建人
  created_at: number;
  updated_at: number;
}

//添加与修改物料
export interface MaterialRequest {
  id: string;
  category_id: string; //物料分类id
  name: string;//物料名称
  image: string;//物料图片
  material: string;//材质：碳钢、不锈钢、合金钢等。
  specification: string;//规格：包括长度、宽度、厚度等尺寸信息。
  model: string;//型号：用于唯一标识和区分不同种类的钢材。
  surface_treatment: string;//表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
  strength_grade: string;//强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
  quantity: number;//安全库存
  unit: string;//计量单位，如个、箱、千克等
  remark: string;//备注
  price: number;//单价
}

//删除物料
export interface MaterialIdRequest {
  id: string;
}

//物料分类列表
export interface MaterialCategorysResponse {
  code: number;
  msg: string;
  // data: MaterialCategoryPaginate;
  data: MaterialCategory[];
}



// export interface MaterialCategoryPaginate {
//   total: number;
//   list: MaterialCategory[];
// }

export interface MaterialCategory {
  id: string; //
  parent_id: string; //上级物料分类id
  sort_id: number; //排序
  name: string; //物料分类名称
  image: string; //物料图片
  status: string; //状态：启用、停用
  remark: string; //备注
  creator_name: string; //创建人
  created_at: string; //
  updated_at: string; //
  children?: MaterialCategory[]; //
}

//添加与修改物料分类
export interface MaterialCategoryRequest {
  id: string; //物料分类Id
  parent_id: string; //上级物料分类Id
  sort_id: number; //排序
  name: string; //物料分类名称
  status: string; //状态：启用 停用
  remark: string; //备注
}

//删除物料分类
export interface MaterialCategoryIdRequest {
  id: string;
}

//物料单价
export interface MaterialPrice {
  price: number;//单价
  since: number; //应用时间
  customer_id: string; //客户id
  customer_name: string; //客户
}

//物料单价列表
export interface MaterialPricesResponse {
  code: number;
  msg: string;
  data: MaterialPrice[];
}

export type QuoteStatus = 'unquoted' | 'quoting' | 'quoted' | 'priced';
export type MaterialQuoteMode = 'detailed' | 'simple';
export type MaterialQuoteStatus = 'draft' | 'submitted' | 'quoted' | 'priced' | 'void';
export type MaterialDeliveryRebuildTaskStatus = 'queued' | 'running' | 'success' | 'failed';

export interface NewCustomerMaterialRequest {
  page: number;
  size: number;
  customer_id: string;
  start_time: number;
  end_time: number;
  quote_status: string;
  material_name: string;
  material_model: string;
}

export interface NewCustomerMaterialExportRequest {
  customer_id: string;
  start_time: number;
  end_time: number;
  quote_status: string;
  material_name: string;
  material_model: string;
}

export interface NewCustomerMaterialItem {
  id: string;
  customer_id: string;
  customer_name: string;
  material_id: string;
  material_name: string;
  material_model: string;
  material_specification: string;
  material_unit: string;
  first_delivery_time: number;
  first_delivery_order_code: string;
  first_delivery_quantity: number;
  first_delivery_price: number;
  quote_status: QuoteStatus;
  latest_quote_id: string;
  latest_quote_no: string;
  latest_price: number;
}

export interface NewCustomerMaterialResponse {
  code: number;
  msg: string;
  data: {
    total: number;
    list: NewCustomerMaterialItem[];
  };
}

export interface MaterialDeliveryRebuildTask {
  id: string;
  status: MaterialDeliveryRebuildTaskStatus | '';
  order_count: number;
  delivery_count: number;
  message: string;
  error_message: string;
  creator_id: string;
  creator_name: string;
  created_at: number;
  started_at: number;
  finished_at: number;
  updated_at: number;
}

export interface MaterialDeliveryRebuildTaskRequest {
  page: number;
  size: number;
  status: string;
}

export interface MaterialDeliveryRebuildTaskResponse {
  code: number;
  msg: string;
  data: MaterialDeliveryRebuildTask;
}

export interface MaterialDeliveryRebuildTaskPageResponse {
  code: number;
  msg: string;
  data: {
    total: number;
    list: MaterialDeliveryRebuildTask[];
  };
}

export interface MaterialQuoteCostItem {
  index: number;
  category_code: string;
  category_name: string;
  name: string;
  enabled: boolean;
  custom: boolean;
  amount: number;
  remark: string;
}

export interface MaterialQuote {
  id: string;
  quote_no: string;
  customer_id: string;
  customer_name: string;
  material_id: string;
  material_name: string;
  material_model: string;
  material_specification: string;
  material_unit: string;
  delivery_id: string;
  source_order_code: string;
  quote_mode: MaterialQuoteMode;
  status: MaterialQuoteStatus;
  currency: string;
  cost_items: MaterialQuoteCostItem[];
  simple_price: number;
  total_cost: number;
  profit_rate: number;
  profit_amount: number;
  tax_rate: number;
  tax_amount: number;
  final_price: number;
  total_amount: number;
  valid_from: number;
  valid_to: number;
  remark: string;
  creator_id: string;
  creator_name: string;
  created_at: number;
  updated_at: number;
}

export interface MaterialQuoteSaveRequest {
  id: string;
  delivery_id: string;
  quote_mode: MaterialQuoteMode;
  currency: string;
  cost_items: MaterialQuoteCostItem[];
  simple_price: number;
  profit_amount: number;
  tax_rate: number;
  final_price: number;
  valid_from: number;
  valid_to: number;
  remark: string;
}

export interface MaterialQuotePageRequest {
  page: number;
  size: number;
  customer_id: string;
  material_id: string;
  delivery_id: string;
  status: string;
  quote_mode: string;
  material_name: string;
  material_model: string;
}

export interface MaterialQuotePageResponse {
  code: number;
  msg: string;
  data: {
    total: number;
    list: MaterialQuote[];
  };
}

export interface MaterialQuoteResponse {
  code: number;
  msg: string;
  data: MaterialQuote;
}

export interface MaterialQuoteIdRequest {
  id: string;
}

export interface MaterialQuotePriceRequest {
  id: string;
  final_price: number;
  effective_at: number;
  remark: string;
}
