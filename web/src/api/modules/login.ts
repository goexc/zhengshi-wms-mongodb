import request from "@/utils/request.ts";
import {PermsResponse} from "@/api/modules/types.ts";
// import authMenuList from "@/assets/json/authMenuList.json";
// import authButtonList from "@/assets/json/authButtonList.json";

/**
 * @name 登录模块
 */
// 用户登录
export const loginApi = (params: any) => {
  return request.post<any, any>(`/auth/login`, params, {  }); // 正常 post json 请求  ==>  application/json
  // return request.post<any, any>(`/auth/login`, params, { noLoading: true }); // 正常 post json 请求  ==>  application/json
  // return request.post<Login.ResLogin>(`/login`, params, { noLoading: true }); // 控制当前请求不显示 loading
  // return request.post<Login.ResLogin>(`/login`, {}, { params }); // post 请求携带 query 参数  ==>  ?username=admin&password=123456
  // return request.post<Login.ResLogin>(`/login`, qs.stringify(params)); // post 请求携带表单参数  ==>  application/x-www-form-urlencoded
  // return request.get<Login.ResLogin>(`/login?${qs.stringify(params, { arrayFormat: "repeat" })}`); // get 请求可以携带数组等复杂参数
};

// 获取菜单列表
export const getAuthMenuListApi = () => {
  return request.get<any, PermsResponse>(
    `/account/menu`,
    {},
    // { noLoading: true },
  );
  // 如果想让菜单变为本地数据，注释上一行代码，并引入本地 authMenuList.json 数据
  // return authMenuList;
};

// // 获取按钮权限
// export const getAuthButtonListApi = () => {
//   return request.get<any, string[]>(`/account/button`, {}, { noLoading: true });
//   // 如果想让按钮权限变为本地数据，注释上一行代码，并引入本地 authButtonList.json 数据
//   // return authButtonList;
// };

// 用户退出登录
export const logoutApi = () => {
  return request.post(`/logout`);
};
