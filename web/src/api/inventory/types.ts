//物料库存列表
export interface InventoryListRequest {
  material_id: string; //物料id
}


export interface InventoryListResponse {
  code: number;
  msg: string;
  data: Inventory[];
}

//物料库存分页
export interface InventorysRequest{
  page: number;
  size: number;
  type: string; //入库单类型：采购入库、外协入库、退货入库
  material_name: string; //物料名称
  material_model: string; //物料型号
  warehouse_id: string; //仓库id
  warehouse_zone_id: string; //库区id
  warehouse_rack_id: string; //货架id
  warehouse_bin_id: string; //货位id
}

export interface InventorysResponse {
  code: number;
  msg: string;
  data: InventorysPaginate;
}


export interface InventorysPaginate {
  total: number;//入库记录数量
  quantity: number; //物料库存总数量
  list: Inventory[];
}

export interface Inventory {
  id: string; //
  type: string; //入库单类型：采购入库、外协入库、退货入库
  entry_time: number; //入库时间
  warehouse_id: string; //仓库id
  warehouse_name: string; //仓库名称
  warehouse_zone_id: string; //库区id
  warehouse_zone_name: string; //库区名称
  warehouse_rack_id: string; //货架id
  warehouse_rack_name: string; //货架名称
  warehouse_bin_id: string; //货位id
  warehouse_bin_name: string; //货位名称
  receipt_code: string; //入库单编号
  receive_code: string; //批次入库编号
  material_id: string; //物料id
  name: string; //物料名称：包括型号、材质、规格、表面处理、强度等级等
  price: number; //物料单价
  model: string; //型号：用于唯一标识和区分不同种类的钢材
  unit: string; //计量单位
  quantity: number; //库存数量
  available_quantity: number; //可用库存数量
  locked_quantity: number; //锁定库存数量
  frozen_quantity: number; //冻结库存数量
  shipment_quantity: number; //出货数量：用于确认出库单
  creator_id: string; //创建人id
  creator_name: string; //创建人名称
  created_at: number; //
}