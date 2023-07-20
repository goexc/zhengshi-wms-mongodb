//api列表
export interface ApiListResponse {
  code: number;
  msg: string;
  data: Api[];
}

export interface Api {
  id: string;
  parent_id: string; //上级apiid
  type: number; //类型：1.模块，2.API
  required: boolean; //是否必选
  sort_id: number; //排序
  uri: string; //请求路径
  method: string; //请求方法,options=|GET|POST|PUT|PATCH|DELETE
  name: string; //api名称
  remark: string; //备注
  created_at?: number;
  updated_at?: number;
  children?: Api[];
}


//添加、修改api
export interface ApiRequest {
  id: string;
  parent_id: string; //上级apiid
  type: number; //api类型：1.api，2.按钮
  sort_id: number; //排序
  name: string;
  path: string; //路由路径
  component: string; //路由组件
  icon: string; //    元信息：图标
  transition: string; //    元信息：过渡动画
  hidden: boolean; //    元信息：是否隐藏
  fixed: boolean; //    元信息：是否固定
  perms: string; //    权限标识
  remark: string;
}

//api状态
export interface ApiStatusRequest {
  id: string[];
  status: string; //api名称
}
