export interface DepartmentListResponse {
  code: number;
  msg: string;
  data: Department[];
}

export interface Department {
  id: string;
  sort_id: string;//排序
  parent_id: string;//上级部门
  name: string;//部门名称
  code: string;//部门编码
  remark: string;//备注
  created_at: string;
  updated_at: string;
  children: Department[];
}

export interface DepartmentRequest {
  id: string;
  sort_id: string;//排序
  parent_id: string;//上级部门
  name: string;//部门名称
  code: string;//部门编码
  remark: string;//备注
}

export interface DepartmentRemoveRequest {
  id: string;
}

