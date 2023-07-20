//库区列表
export interface ZonesRequest {
  page: number;//页数
  size: number;//条数
  warehouse_id: string; //仓库
  name?: string; //库区名称
  code?: string; //库区编号
  status?: string; //库区状态:激活 禁用 盘点中 关闭
}

export interface ZonesResponse {
  code: number;
  msg: string;
  data: ZonePaginate;
}

export interface ZonePaginate {
  total: number;
  list: Zone[];
}

export interface Zone {
  id: string; //新增库区没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  name: string; //库区名称
  code: string; //库区编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  capacity: number; //库区容量
  capacity_unit: string; //库区容量单位：面积、体积或其他度量单位
  status: string; //库区状态
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
  create_by?: string; //新增库区没有create_by，创建人
  created_at?: number; //
  updated_at?: number; //
}

//添加、修改库区
export interface ZoneRequest {
  id: string; //库区id
  warehouse_id: string; //仓库Id
  name: string; //库区名称
  code: string; //库区编号
  image: string; //图片链接
  // status: string; //库区状态：激活 禁用 盘点中 关闭 删除
  capacity: number; //库区容量
  capacity_unit: string; //库区容量单位
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
}


//修改库区状态[包括删除]
export interface ZoneStatusRequest {
  id: string;
  status: string; //库区状态：激活 禁用 盘点中 关闭 删除
}