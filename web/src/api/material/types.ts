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
