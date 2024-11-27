//登录接口需要携带的参数类型
export interface loginForm {
  mobile: string;
  password: string;
}

export interface dataType {
  token: string;
}

//登录接口返回数据类型
export interface loginResponse {
  code: number;
  msg: string;
  data: dataType;
}

export interface userInfo {
  userId: number;
  avatar: string;
  name: string;
  desc: string;
  roles: string[];
  buttons: string[];
  routes: string[];
  token: string;
}

// export interface role {
//   role_id: string;
//   role_name: string;
// }

// export interface button {
//   name: string;
//   icon: string;
//   perms: string;
// }

export interface Account {
  name: string;
  avatar: string;
  desc: string;
  // roles: role[];
  // buttons: button[];
  routes: string[];
}

//定义用户信息数据类型
export interface accountInfoResponse {
  code: number;
  msg: string;
  data: Account;
}

//用户的菜单id列表
export interface accountMenusResponse {
  code: number;
  msg: string;
  data: string[]; //菜单id列表
}
