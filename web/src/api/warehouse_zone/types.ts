//库区列表
export interface WarehouseZoneListRequest {
  warehouse_id: string; //仓库
  type?: string; //库区类型:分销中心 生产库区 跨境库区 电商库区 冷链库区 合规库区 专用库区 跨渠道库区 自动化库区 第三方物流库区
  name?: string; //库区名称
  code?: string; //库区编号
  status?: string; //库区状态:激活 禁用 盘点中 关闭
}

export interface WarehouseZoneListResponse {
  code: number;
  msg: string;
  data: WarehouseZonePaginate;
}

export interface WarehouseZonePaginate {
  total: number;
  list: WarehouseZone[];
}

export interface WarehouseZone {
  id: string; //新增库区没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  name: string; //库区名称
  type: string; //库区类型:分销中心 生产库区 跨境库区 电商库区 冷链库区 合规库区 专用库区 跨渠道库区 自动化库区 第三方物流库区
  code: string; //库区编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  capacity: number; //库区容量
  capacity_unit: string; //库区容量单位：面积、体积或其他度量单位
  status: string; //库区状态
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
  create_by?: string; //新增库区没有create_by，创建人
  created_at: number; //
  updated_at: number; //
}
