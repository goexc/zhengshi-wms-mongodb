export interface InboundReceiptsRequest {
  page:number;
  size:number;
  status: string; //入库单状态
  type: string; //入库单类型
  code: string; //入库单号
  supplier_id: string; //供应商
  customer_id: string; //客户
}

export interface InboundReceiptsResponse {
  code: number;
  msg: string;
  data: InboundReceiptPaginate;
}

export interface InboundReceiptPaginate {
  total: number;
  list: InboundReceipt[];
}

export interface InboundReceipt {
  id: string; //新增入库单没有id
  code: string; //入库单编号：
  // 入库单类型
  //采购入库
  //外协入库
  //退货入库
  type: string; //入库单类型：
  //入库单状态
  //待审核：入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
  //审核不通过：入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
  //审核通过：入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
  //未发货：
  //在途：
  //部分入库：当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
  //作废：当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
  //入库完成：当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
  status: string; //入库单状态:
  total_amount: number; //总金额:
  supplier_id: string,//供应商：适用于采购入库、退货入库等类型
  customer_id: string,//客户：适用于退货类型
  receiving_date: number,//入库日期
  materials: InboundMaterial[],//物料
  annex: string[], //附件
  remark: string; //备注：
  created_at?: number; //
  updated_at?: number; //
}

export interface InboundMaterial {
  index: number;//物料顺序
  id: string;//物料
  price: number;//单价
  estimated_quantity: number;//预计入库数量
  actual_quantity: number;//实际入库数量
  positions: string[]; //仓储id：仓库id，库区id，货架id，货位id
  warehouse_id: string;//仓库
  warehouse_zone_id: string;//库区
  warehouse_rack_id: string;//货架
  warehouse_bin_id: string;//货位
}

//添加、修改入库单
export interface InboundReceiptRequest {
  id: string; //新增入库单没有id
  code: string; //入库单编号：
  // 入库单类型
  //采购入库
  //外协入库
  //退货入库
  type: string; //入库单类型：
  //入库单状态
  //待审核：入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
  //审核不通过：入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
  //审核通过：入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
  //未发货：
  //在途：
  //部分入库：当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
  //作废：当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
  //入库完成：当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
  status: string; //入库单状态:
  supplier_id: string,//供应商
  receiving_date: number,//入库日期
  materials: InboundMaterial[],//物料
  annex: string[], //附件
  remark: string; //备注：
}

//修改入库单状态[包括删除]
export interface InboundReceiptStatusRequest {
  id: string;
  status: string; //入库单状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
}

//审核入库单
export interface InboundReceiptCheckRequest {
  id: string;
  status: string; //入库单状态：审核通过，审核不通过
}

//物料入库
export interface InboundReceiptMaterialRequest {
  id: string;
  total_amount: number; //总金额
  materials: InboundMaterialStatus[];
}

//物料状态
export interface InboundMaterialStatus {
  id: string;
  status: string;
  actual_quantity: number;
}