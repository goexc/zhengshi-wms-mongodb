export interface CustomersRequest {
  page:number;
  size:number;
  name: string; //客户名称
  code: string; //客户编号
  manager: string; //联系人
  contact: string; //联系方式
  email: string; //邮箱
  level: number; //客户等级
}

export interface CustomersResponse {
  code: number;
  msg: string;
  data: CustomerPaginate;
}

export interface CustomerPaginate {
  total: number;
  list: Customer[];
}

export interface Customer {
  id: string; //新增客户没有id
  type: string; //客户类型：个人、企业、组织
  name: string; //客户名称
  level: number; //客户等级
  code: string; //客户编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片链接
  legal_representative: string; //法定代表人
  unified_social_credit_identifier: string; //统一社会信用代码
  address: string; //客户地址
  manager: string; //负责人
  contact: string; //联系方式
  status: string; //客户状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
  remark: string; //备注
  create_by?: string; //新增客户没有create_by，创建人
  created_at?: number; //
  updated_at?: number; //
}

//添加、修改客户
export interface CustomerRequest {
  id: string; //新增客户没有id
  type: string; //客户类型：个人、企业、组织
  name: string; //客户名称
  code: string; //客户编号：分配给客户的唯一标识符或编号，用于快速识别和检索客户信息
  image: string; //图片链接
  legal_representative: string; //法定代表人
  unified_social_credit_identifier: string; //统一社会信用代码
  address: string; //客户地址
  manager: string; //负责人
  contact: string; //联系方式
  remark: string; //备注
}

//修改客户状态[包括删除]
export interface CustomerStatusRequest {
  id: string;
  status: string; //客户状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
}
