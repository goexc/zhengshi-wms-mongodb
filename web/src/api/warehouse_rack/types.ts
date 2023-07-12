//货架列表
export interface WarehouseRackListRequest {
  warehouse_id: string; //仓库
  warehouse_zone_id: string; //库区
  type?: string; //货架类型:标准货架 重型货架 中型货架 轻型货架
  name?: string; //货架名称
  code?: string; //货架编号
  status?: string; //货架状态:激活 禁用 盘点中 关闭
}

export interface WarehouseRackListResponse {
  code: number;
  msg: string;
  data: WarehouseRackPaginate;
}

export interface WarehouseRackPaginate {
  total: number;
  list: WarehouseRack[];
}

export interface WarehouseRack {
  id: string; //新增库区没有id
  warehouse_id: string; //仓库Id
  warehouse_name: string; //仓库名称
  warehouse_zone_id: string; //库区Id
  warehouse_zone_name: string; //库区名称
  name: string; //货架名称
  type: string; //货架类型:标准货架 重型货架 中型货架 轻型货架
  code: string; //货架编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  capacity: number; //货架容量
  capacity_unit: string; //货架容量单位：面积、体积或其他度量单位
  status: string; //货架状态:激活 禁用 盘点中 关闭
  remark: string; //备注
  create_by: string; //新增货架没有create_by，创建人
  created_at: number; //
  updated_at: number; //
}
