//用户分页
export interface UsersRequest {
  page: number; //页数
  size: number; //条数
  name: string; //用户名
  mobile: string; //手机号码
}

//
export interface UsersResponse {
  code: number;
  msg: string;
  data: UserPaginate;
}

export interface UserPaginate {
  total: number;
  list: User[];
}

export interface User {
  id: string;
  name: string;
  sex: string;
  department_id: string;
  department_name: string;
  roles_id: string[];
  roles_name: string[];
  mobile: string;
  email: string;
  status: string;
  remark: string;
  created_at: number;
  updated_at: number;
}

//添加或修改用户
export interface UserRequest {
  id: string; //账号id
  name: string; //账号名称
  password: string; //账号密码
  sex: string; //性别
  department_id: string; //部门
  roles_id: string[]; //角色
  mobile: string; //手机号码
  email: string; //Email
  status: string; //状态
  remark: string; //备注
}

//用户状态
export interface UserStatusRequest {
  id: string[]; //用户id
  status: string; //用户状态：启用，禁用，删除
}

//用户角色
export interface UserRolesRequest {
  id: string; //用户id
  roles_id: string[]; //角色id
}
