export interface OutboundOrdersRequest {
  page: number;
  size: number;
  status: string; //出库单状态
  is_pack: number; //是否打包：-1忽略，0否,1是
  is_weigh: number; //是否称重：-1忽略，0否,1是
  type: string; //出库单类型
  code: string; //出库单号
  supplier_id: string; //供应商
  customer_id: string; //客户
  start_time: number; //签收起始日期
  end_time: number; //签收截止日期
}

export interface OutboundOrdersResponse {
  code: number;
  msg: string;
  data: OutboundOrderPaginate;
}

export interface OutboundOrderPaginate {
  total: number;
  list: OutboundOrder[];
}

export interface OutboundOrder {
  id: string; //新增出库单没有id
  code: string; //出库单号：
  // 出库单类型
  //采购出库
  //外协出库
  //退货出库
  type: string; //出库单类型：
  //出库单状态
  //待审核：出库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对出库单进行审核。
  //审核不通过：出库单未通过审核时的状态，通常需要重新修改或撤销出库单。
  //审核通过：出库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
  //未发货：
  //在途：
  //部分出库：当出库单中的部分物料已出库，但尚有未出库的物料时，状态为部分出库。
  //作废：当出库单发生错误或不再需要时，可以将其状态设置为作废，表示该出库单无效。
  //出库完成：当出库单中的所有物料都已经成功出库并完成相关操作时，状态为出库完成。
  status: string; //出库单状态:
  is_weigh:number; //是否称重
  is_pack:number; //是否打包
  total_amount: number; //总金额:
  supplier_id: string,//供应商：适用于采购出库、退货出库等类型
  customer_id: string,//客户：适用于退货类型
  customer_name: string,//客户名称
  carrier_name: string,//承运商
  carrier_cost: number,//运费
  other_cost: number,//其他费用
  date: number,//出库日期：×
  departure_time: number,//出库日期
  receipt_time: number,//签收日期
  materials: OutboundOrderMaterial[], //物料列表
  annex: string[], //附件
  // receipt: string[], //收据
  remark: string; //备注：
  created_at?: number; //
  updated_at?: number; //
}

//添加出库单
export interface OutboundOrderRequest {
  id: string; //新增出库单没有id
  code: string; //出库单号：
  // 出库单类型
  //采购出库
  //外协出库
  //退货出库
  type: string; //出库单类型：
  //出库单状态
  //待审核：出库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对出库单进行审核。
  //审核不通过：出库单未通过审核时的状态，通常需要重新修改或撤销出库单。
  //审核通过：出库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
  //未发货：
  //在途：
  //部分出库：当出库单中的部分物料已出库，但尚有未出库的物料时，状态为部分出库。
  //作废：当出库单发生错误或不再需要时，可以将其状态设置为作废，表示该出库单无效。
  //出库完成：当出库单中的所有物料都已经成功出库并完成相关操作时，状态为出库完成。
  status: string; //出库单状态:
  supplier_id: string,//供应商
  date: number,//出库日期
  annex: string[], //附件
  remark: string; //备注：
}

//确认出库单
export interface OutboundOrderConfirmRequest {
  code: string; //出库单号
  confirm_time: number; //确认出库单时间
  materials: OutboundConfirmMaterial[];
}

export interface OutboundConfirmMaterial {
  material_id: string; //物料id
  index: number; //物料序号
  inventorys: OutboundConfirmMaterialInventory[]; //出库库存
}

export interface OutboundConfirmMaterialInventory {
  inventory_id: string; //库存id
  shipment_quantity: number; //出货数量
}

//审核出库单
export interface OutboundOrderCheckRequest {
  id: string;
  status: string; //出库单状态：审核通过，审核不通过
}

//获取出库单物料列表
export interface OutboundOrderMaterialsRequest {
  order_code: string;//出库单号
}

export interface OutboundOrderMaterialsResponse {
  code: number;
  msg: string;
  data: OutboundOrderMaterial[];//物料列表
}

export interface OutboundOrderMaterial {
  id: string;//出库单详情表id
  index: number;//物料顺序
  material_id: string;//物料id
  name: string;//物料名称
  model: string;//物料型号
  specification: string;//物料规格
  price: number;//单价
  quantity: number;//出库数量
  unit: string;//计量单位
  weight: number; //重量（kg）
  returned_quantity: number; //退货数量
}

//物料状态
export interface OutboundOrderMaterialStatus {
  id: string;
  status: string;
  actual_quantity: number;
}

//添加批次出库
export interface OutboundOrderReceiveRequest {
  id: string, //出库单
  code: string; //批次出库编号
  // date: number | string, //批次出库日期
  carrier_id: string, //承运商
  carrier_cost: number, //运费
  other_cost: number, //其他费用
  materials: OutboundReceived[], //物料
  remark: string, //备注
}

export interface OutboundReceived {
  id: string, //物料id
  index: number, //物料顺序
  price: number, //单价
  actual_quantity: number, //实际出库数量
  position: string[], //仓储位置id
  status: string, //物料状态: 未发货，部分出库,作废,出库完成
}

//拣货
export interface OutboundPickingRequest {
  code: string; //出库单号
  materials: OutboundPickingMaterial[];
}

export interface OutboundPickingMaterial {
  material_id: string; //物料id
  index: number; //物料序号
  // price: number; //物料单价
  inventorys: OutboundPickingMaterialInventory[]; //出库库存
}

export interface OutboundPickingMaterialInventory {
  inventory_id: string; //库存id
  shipment_quantity: number; //出货数量
}

//出库单拣货
export interface OutboundOrderPickRequest {
  code: string; //出库单号
  picking_time: number; //确认拣货时间
}

//出库单打包
export interface OutboundOrderPackRequest {
  code: string; //出库单号
  packing_time: number; //确认打包时间
}

//称重
export interface OutboundOrderWeighRequest {
  code: string; //出库单号
  weighing_time: number; //称重时间
  materials: OutboundWeighMaterial[];
}

export interface OutboundWeighMaterial {
  material_id: string; //物料id
  weight: number; //重量
}

//出库
export interface OutboundOrderDepartureRequest {
  code: string; //出库单号
  departure_time: number; //出库时间
  has_carrier: boolean; //是否承运
  carrier_id: string; //承运商id
  carrier_cost: number; //运费
  other_cost: number; //其他费用
}

//签收
export interface OutboundOrderReceiptRequest {
  code: string; //出库单号
  annex: string[]; //附件
  receipt_time: number; //签收时间
}


//出库汇总
export interface OutboundOrderSummaryRequest {
  customer_id: string; //客户
  start_date: number; //起始日期
  end_date: number; //截止日期
}

export interface OutboundOrderSummaryResponse {
  code: number;
  msg: string;
  data: OutboundOrderRecord[];
}

export interface OutboundOrderRecord extends OutboundOrderMaterial {
  code: string;//出库单号
  departure_date: number;//出库日期
  receipt_date: number;//签收日期
  order_code: string;//出库单编号
}

//修改出库单物料单价
export interface OutboundOrderMaterialRevise {
  code: string; //出库单号
  customer_id: string; //客户id
  materials_price: OutboundMaterialPrice[]; //物料单价
}

export interface OutboundMaterialPrice {
  material_id: string; //物料id
  price: number; //重量
}