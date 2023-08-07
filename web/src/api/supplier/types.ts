export interface SuppliersRequest {
  page:number;
  size:number;
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
