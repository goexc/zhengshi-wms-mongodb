export interface RoleIdRequest {
  id: string; //角色id
}

//角色列表
export interface RoleListRequest {
  name: string; //角色名称
}

export interface RoleListResponse {
  code: number;
  msg: string;
  data: RolePaginate;
}

//角色分页
export interface RolesRequest {
  page: number; //页数
  size: number; //条数
  name: string; //角色名称
}

export interface RolesResponse {
  code: number;
  msg: string;
  data: RolePaginate;
}

export interface RolePaginate {
  total: number;
  list: Role[];
}

export interface Role {
  id: string;
  parent_id: string; //上级角色id
  name: string; //角色名称
  status: string; //状态：停用，启用，删除
  remark: string; //备注
  created_at: number;
  updated_at: number;
}

//添加、修改角色
export interface RoleRequest {
  id: string;
  parent_id: string; //上级角色id
  status: string; //状态：停用，启用，删除
  name: string;
  remark: string;
}

//角色状态
export interface RoleStatusRequest {
  id: string[];
  status: string; //角色名称
}

//分配角色菜单
export interface RoleMenusRequest {
  id: string;
  menus_id: string[]; //菜单id
}

//角色菜单列表
export interface RoleMenusResponse {
  code: number;
  msg: string;
  data: string[]; //menu_id数组
}
