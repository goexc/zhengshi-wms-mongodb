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
  email: string; //email
  status: string; //客户状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
  remark: string; //备注
  receivable_balance: number; //应收账款余额
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
  receivable_balance: number; //应收账款余额
}

//修改客户状态[包括删除]
export interface CustomerStatusRequest {
  id: string;
  status: string; //客户状态:10.审核中;20.审核不通过;30.活动;40.停用;50.黑名单;60.合同到期;100.删除
}

//客户交易流水
export interface CustomerTransactionPageRequest {
  page: number;
  size: number;
  customer_id: string;
}

export interface CustomerTransactionsResponse {
  code: number;
  msg: string;
  data: CustomerTransactionPaginate;
}

export interface CustomerTransactionPaginate {
  total: number;  //交易流水条数
  list: CustomerTransaction[];
}

export interface CustomerTransaction {
  type: string;//交易类型：应收账款、回款、退货
  time: number; //交易时间
  amount: number; //交易金额
  remark: number; //备注
  annex: string; //附件
}

//添加客户交易记录
export interface CustomerTransactionAddRequest {
  customer_id: string;//客户id
  time: number;//交易类型：应收账款、回款、退货
  type: string;//交易时间
  amount: number;//交易金额
  remark: string;//备注
  annex: string[]; //附件
}