//货架列表
export interface RackListRequest {
  warehouse_id: string; //仓库
  warehouse_zone_id: string; //货架
  type?: string; //货架类型:标准货架 重型货架 中型货架 轻型货架
  name?: string; //货架名称
  code?: string; //货架编号
  status?: string; //货架状态:激活 禁用 盘点中 关闭
}

export interface RackListResponse {
  code: number;
  msg: string;
  data: RackPaginate;
}

export interface RackPaginate {
  total: number;
  list: Rack[];
}

export interface Rack {
  id: string; //新增货架没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  warehouse_zone_id: string; //货架Id
  warehouse_zone_name: string; //货架名称
  name: string; //货架名称
  type: string; //货架类型:标准货架 重型货架 中型货架 轻型货架
  code: string; //货架编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片
  capacity: number; //货架容量
  capacity_unit: string; //货架容量单位：面积、体积或其他度量单位
  status: string; //货架状态:激活 禁用 盘点中 关闭
  remark: string; //备注
  create_by: string; //新增货架没有create_by，创建人
  created_at: number; //
  updated_at: number; //
}


//添加、修改货架
export interface RackRequest {
  id: string; //货架id
  warehouse_id: string; //仓库Id
  warehouse_zone_id: string; //库区
  name: string; //货架名称
  code: string; //货架编号
  image: string; //图片链接
  capacity: number; //货架容量
  capacity_unit: string; //货架容量单位
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
}

//货架列表
export interface RacksRequest {
  page: number;//页数
  size: number;//条数
  warehouse_id: string; //仓库
  warehouse_zone_id: string; //库区
  name?: string; //货架名称
  type?: string; //货架类型
  code?: string; //货架编号
  status?: string; //货架状态:激活 禁用 盘点中 关闭
}

export interface RacksResponse {
  code: number;
  msg: string;
  data: RackPaginate;
}

export interface RackPaginate {
  total: number;
  list: Rack[];
}

export interface Rack {
  id: string; //新增货架没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  name: string; //货架名称
  code: string; //货架编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  capacity: number; //货架容量
  capacity_unit: string; //货架容量单位：面积、体积或其他度量单位
  status: string; //货架状态
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
  create_by: string; //新增货架没有create_by，创建人
  created_at: number; //
  updated_at: number; //
}


//修改货架状态[包括删除]
export interface RackStatusRequest {
  id: string;
  status: string; //货架状态：激活 禁用 盘点中 关闭 删除
}