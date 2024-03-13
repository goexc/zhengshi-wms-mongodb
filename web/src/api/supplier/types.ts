export interface SuppliersRequest {
  page: number;
  size: number;
  name: string; //供应商名称
  code: string; //供应商编号
  manager: string; //联系人
  contact: string; //联系方式
  email: string; //邮箱
  level: number; //供应商等级
}

export interface SuppliersResponse {
  code: number;
  msg: string;
  data: SupplierPaginate;
}

export interface SupplierPaginate {
  total: number;
  list: Supplier[];
}

export interface Supplier {
  id: string; //新增供应商没有id
  type: string; //供应商类型：个人、企业、组织
  name: string; //供应商名称
  level: number; //供应商等级
  code: string; //供应商编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片链接
  legal_representative: string; //法定代表人
  unified_social_credit_identifier: string; //统一社会信用代码
  address: string; //供应商地址
  manager: string; //负责人
  contact: string; //联系方式
  email: string; //邮箱
  status: string; //供应商状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
  remark: string; //备注
  create_by?: string; //新增供应商没有create_by，创建人
  created_at?: number; //
  updated_at?: number; //
}

//添加、修改供应商
export interface SupplierRequest {
  id: string; //新增供应商没有id
  type: string; //供应商类型：个人、企业、组织
  name: string; //供应商名称
  code: string; //供应商编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片链接
  legal_representative: string; //法定代表人
  unified_social_credit_identifier: string; //统一社会信用代码
  address: string; //供应商地址
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
}

//修改供应商状态[包括删除]
export interface SupplierStatusRequest {
  id: string;
  status: string; //供应商状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
}

//供应商订单分页
export interface SupplierOrdersRequest {
  page: number;//页码
  size: number; //每页条数
  supplier_id: string; //供应商id
  order_code: string; //订单编号
  start_date: number; //开始日期
  end_date: number; //结束日期
}

export interface SupplierOrdersResponse {
  code: number;
  msg: string;
  data: SupplierOrderPaginate;
}

export interface SupplierOrderPaginate {
  total: number;
  list: SupplierOrder[];
}

export interface SupplierOrder {
  id: string;
  code: string;
  type: string;//订单类型：
  status: string;//订单状态
  estimated_total_amount: number;//预计总金额
  total_amount: number;//总金额
  supplier_id: string;//供应商id
  supplier_name: string;//供应商名称
  materials: SupplierOrderMaterial[];
  remark: string;
  created_at: number;//创建时间
  creator_id: string;//创建人
  creator_name: string;//创建人
}

export interface SupplierOrderMaterial {
  id: string; //物料id
  name: string;//物料名称
  code: string;//物料编号
  model: string;//物料型号
  specification: string;//物料规格
  price: number;//物料单价
  estimated_quantity: number;//采购数量
  quantity: number;//到货数量
  delivery_time:number;//到货时间
}