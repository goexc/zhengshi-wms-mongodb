//货位列表
export interface WarehouseBinListRequest {
  warehouse_id: string; //仓库
  warehouse_zone_id: string; //库区
  warehouse_rack_id: string; //货架
  name?: string; //货位名称
  code?: string; //货位编号
  status?: string; //货位状态:激活 禁用 盘点中 关闭
}

export interface WarehouseBinListResponse {
  code: number;
  msg: string;
  data: WarehouseBinPaginate;
}
//货位分页
export interface WarehouseBinsRequest {
  page: number;
  size: number;
  warehouse_id: string; //仓库
  warehouse_zone_id: string; //库区
  warehouse_rack_id: string; //货架
  name?: string; //货位名称
  code?: string; //货位编号
  status?: string; //货位状态:激活 禁用 盘点中 关闭
}

export interface WarehouseBinsResponse {
  code: number;
  msg: string;
  data: WarehouseBinPaginate;
}

export interface WarehouseBinPaginate {
  total: number;
  list: WarehouseBin[];
}

export interface WarehouseBin {
  id: string; //新增库区没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  warehouse_zone_id: string; //库区Id
  warehouse_zone_name: string; //库区名称
  warehouse_rack_id: string; //货架Id
  warehouse_rack_name: string; //货架名称
  name: string; //货位名称
  code: string; //货位编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片
  capacity: number; //货位容量
  capacity_unit: string; //货位容量单位：面积、体积或其他度量单位
  status: string; //货位状态:激活 禁用 盘点中 关闭
  manager: string; //联系人
  contact: string; //联系方式
  remark: string; //备注
  create_by: string; //新增货位没有create_by，创建人
  created_at: number; //
  updated_at: number; //
}

//添加或修改货位
export interface WarehouseBinRequest {
  id: string; //货位id
  warehouse_rack_id: string; //货架id
  name: string; //货位名称
  code: string; //货位编号
  image: string; //图片链接
  capacity: number; //货位容量
  capacity_unit: string; //货位容量单位
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
}

//修改货位状态
export interface WarehouseBinStatusRequest {
  id: string;
  status: string; //货位状态：激活 禁用 盘点中 关闭 删除
}
