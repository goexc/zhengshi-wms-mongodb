//仓库树
export interface WarehouseTreeResponse {
  code: number;
  msg: string;
  data: WarehouseTree[];
}

export interface WarehouseTree {
  id:string;
  name: string;
  children: WarehouseTree[];
}

//仓库分页
export interface WarehousesRequest {
  page: number;//页数
  size: number;//条数
  type?: string;//仓库类型
  name?: string;//仓库名称
  code?: string;//仓库编号
  status?: string;//仓库状态
}

export interface Warehouse {
  id: string; //新增仓库没有id
  type: string; //仓库类型
  name: string; //仓库名称
  code: string; //仓库编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  address: string; //仓库地址
  capacity: number; //仓库容量
  capacity_unit: string; //仓库容量单位：面积、体积或其他度量单位
  // status: string; //仓库状态
  manager: string; //负责人
  contact: string; //联系方式
  image: string; //图片链接
  remark: string; //备注
  create_by?: string; //新增仓库没有create_by，创建人
  created_at?: number; //
  updated_at?: number; //
}

export interface WarehousesResponse {
  code: number;
  msg: string;
  data: WarehousePaginate;
}

export interface WarehousePaginate {
  total: number;
  list: Warehouse[];
}

//添加、修改仓库
export interface WarehouseRequest {
  id: string; //仓库id
  type: string; //仓库类型
  name: string; //仓库名称
  code: string; //仓库编号
  address: string; //仓库地址
  capacity: number; //仓库容量
  capacity_unit: string; //仓库容量单位
  manager: string; //负责人
  contact: string; //联系方式
  image: string; //图片链接
  remark: string; //备注
}

export interface WarehouseResponse {
  code: number;
  msg: string;
  data: WarehouseUrl;
}

export interface WarehouseUrl {
  url: string;
}

//修改仓库状态[包括删除]
export interface WarehouseStatusRequest {
  id: string;
  status: string; //仓库状态：激活 禁用 盘点中 关闭 删除
}
