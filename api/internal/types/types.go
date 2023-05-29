// Code generated by goctl. DO NOT EDIT.
package types

type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type LoginRequest struct {
	Account  string `json:"account"`  //账号名称
	Password string `json:"password"` //账号密码
}

type LoginResponse struct {
	Code int       `json:"code"`
	Msg  string    `json:"msg"`
	Data LoginData `json:"data,optional"`
}

type LoginData struct {
	Account string `json:"account,optional"` //账号名称
	Token   string `json:"token,optional"`   //Token
	Exp     int64  `json:"exp,optional"`     //过期时间戳
}

type ApiAddRequest struct {
	Type     int64  `json:"type,options=1|2"`                          //类型：1.模块，2.API
	SortId   int64  `json:"sort_id,range=[0:]"`                        //排序
	ParentId string `json:"parent_id"`                                 //上级id
	Uri      string `json:"uri,optional"`                              //请求路径
	Method   string `json:"method,options=|GET|POST|PUT|PATCH|DELETE"` //请求方法
	Name     string `json:"name"`                                      //名称
	Remark   string `json:"remark,optional"`                           //备注
}

type ApiUpdateRequest struct {
	Api
}

type ApiIdRequest struct {
	Id string `form:"id"`
}

type ApisResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []Api  `json:"data"`
}

type Api struct {
	Id       string `json:"id"`                                        //
	Type     int64  `json:"type,options=1|2"`                          //类型：1.模块，2.API
	SortId   int64  `json:"sort_id,range=[0:]"`                        //排序
	ParentId string `json:"parent_id"`                                 //上级id
	Uri      string `json:"uri,optional"`                              //请求路径
	Method   string `json:"method,options=|GET|POST|PUT|PATCH|DELETE"` //请求方法
	Name     string `json:"name"`                                      //名称
	Remark   string `json:"remark,optional"`                           //备注
}

type MenuRemoveRequest struct {
	Id string `form:"id":"id"`
}

type MenuRequest struct {
	Id         string `json:"id,optional"`
	Type       int64  `json:"type,options=1|2"`   //路由类型：1.菜单，2.按钮
	SortId     int64  `json:"sort_id"`            //排序
	ParentId   string `json:"parent_id"`          //父路由id
	Path       string `json:"path,optional"`      //路由路径
	Name       string `json:"name,optional"`      //路由名称：如，System
	Component  string `json:"component,optional"` //路由组件
	Icon       string `json:"icon"`               //图标
	Transition string `json:"transition"`         //过渡动画
	Hidden     bool   `json:"hidden"`             //是否隐藏
	Fixed      bool   `json:"fixed"`              //是否固定
	Perms      string `json:"perms,optional"`     //权限标识
	Remark     string `json:"remark"`             //备注
}

type MenusResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []Menu `json:"data"`
}

type Menu struct {
	Id         string `json:"id"`
	Type       int64  `json:"type"`               //路由类型：1.菜单，2.按钮
	SortId     int64  `json:"sort_id"`            //排序
	ParentId   string `json:"parent_id"`          //父路由id
	Path       string `json:"path,optional"`      //路由路径
	Name       string `json:"name,optional"`      //路由名称
	Component  string `json:"component,optional"` //路由组件
	Icon       string `json:"icon"`               //元信息：图标
	Transition string `json:"transition"`         //元信息：过渡动画
	Hidden     bool   `json:"hidden"`             //元信息：是否隐藏
	Fixed      bool   `json:"fixed"`              //元信息：是否固定
	Perms      string `json:"perms,optional"`     //权限标识
	Remark     string `json:"remark"`             //备注
	Children   []Menu `json:"children,optional"`
}

type RoleIdRequest struct {
	Id string `form:"id"`
}

type RoleRequest struct {
	Id       string `json:"id,optional"`
	ParentId string `json:"parent_id"`          //上级角色id
	SortId   int64  `json:"sort_id,range=[0:]"` //排序
	Status   int64  `json:"status,range=(0:]"`  //状态：10.停用，20.在用
	Name     string `json:"name"`               //角色名称
	Remark   string `json:"remark"`             //备注
}

type RoleListRequest struct {
	Page int64  `form:"page,range=[1:]"`
	Size int64  `form:"size,range=[1:]"`
	Name string `form:"name,optional"` //角色名称
}

type RoleListResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data RolePaginate `json:"data"`
}

type RolePaginate struct {
	Total int64  `json:"total,optional"` //用于返回全部角色时，不可用
	List  []Role `json:"list"`           //角色分页
}

type Role struct {
	Id        string `json:"id"`
	ParentId  string `json:"parent_id"`          //上级角色id
	SortId    int64  `json:"sort_id,range=[0:]"` //排序
	Status    int64  `json:"status,range=(0:]"`  //状态：10.停用，20.在用
	Name      string `json:"name"`               //角色名称
	Remark    string `json:"remark"`             //备注
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type RoleStatusRequest struct {
	Id     string `form:"id"`
	Status int64  `form:"status,range=(0:]"` //状态：10.停用，20.在用
}

type RoleMenusRequest struct {
	RoleId  string   `json:"role_id"`  //角色id
	MenusId []string `json:"menus_id"` //菜单id
}

type RoleMenusResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data []string `json:"data"`
}

type RoleApisRequest struct {
	RoleId string   `json:"role_id"` //角色id
	ApisId []string `json:"apis_id"` //api id
}

type RoleApisResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data []string `json:"data"`
}

type DepartmentRemoveRequest struct {
	Id string `form:"id"`
}

type DepartmentResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data Department `json:"data,optional"`
}

type Department struct {
	Id        string `json:"id,optional" path:"id"`
	Type      int64  `json:"type"`               //部门类型：20.小组，40.部门，60.子公司，80.公司
	SortId    int64  `json:"sort_id"`            //排序
	ParentId  string `json:"parent_id,optional"` //上级部门
	Name      string `json:"name"`               //部门名称
	FullName  string `json:"full_name"`          //部门全称
	Code      string `json:"code"`               //部门编码
	Remark    string `json:"remark"`             //备注
	CreatedAt int64  `json:"created_at,optional"`
	UpdatedAt int64  `json:"updated_at,optional"`
}

type DepartmentsResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data []Department `json:"data"` //部门列表
}

type DepartmentRequest struct {
	Id        string `json:"id,optional"`
	Type      int64  `json:"type"`               //部门类型：20.小组，40.部门，60.子公司，80.公司
	SortId    int64  `json:"sort_id"`            //排序
	ParentId  string `json:"parent_id,optional"` //上级部门
	Name      string `json:"name"`               //部门名称
	FullName  string `json:"full_name"`          //部门全称
	Code      string `json:"code"`               //部门编码
	Remark    string `json:"remark"`             //备注
	CreatedAt int64  `json:"created_at,optional"`
	UpdatedAt int64  `json:"updated_at,optional"`
}

type UserIdRequest struct {
	Id int64 `form:"id,range=[1:]"`
}

type UserRequest struct {
	User
}

type ResetUserPasswordRequest struct {
	Id       int64  `json:"id"`
	Password string `json:"password"`
}

type UserStatusRequest struct {
	Id     int64 `json:"id"`
	Status int64 `json:"status"` //状态：0.禁用，10.启用
}

type UserListRequest struct {
	Page   int    `form:"page,range=[1:]"`
	Size   int    `form:"size,range=[1:]"`
	Name   string `form:"name,optional"`   //搜索关键词：用户名
	Mobile string `form:"mobile,optional"` //搜索关键词：手机号码
}

type UserListResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data UserPaginate `json:"data"`
}

type UserPaginate struct {
	Total int64  `json:"total"`
	List  []User `json:"list"` //用户列表
}

type User struct {
	Id             string   `json:"id,optional"`
	Account        string   `json:"account" validate:"required" comment:"账号名称"` //账号名称
	Password       string   `json:"password"`                                   //用户密码
	Sex            int      `json:"sex,options=0|1|2"`                          //性别：0.女，1.男，2.未知
	DepartmentId   string   `json:"department_id"`                              //部门id
	DepartmentName string   `json:"department_name,optional"`                   //部门名称
	RolesId        []string `json:"roles_id"`                                   //角色id
	Mobile         string   `json:"mobile"`                                     //手机号码
	Email          string   `json:"email,optional" validate:"email"`            //邮箱
	Status         int64    `json:"status,optional"`                            //用户状态：0.未启用，20.启用，50.禁用
	Remark         string   `json:"remark,optional"`                            //备注
	CreatedAt      int64    `json:"created_at,optional"`
	UpdatedAt      int64    `json:"updated_at,optional"`
}

type WarehouseIdRequest struct {
	Id string `form:"id":"id"`
}

type WarehouseRequest struct {
	Id      string  `json:"id,optional"`
	Name    string  `json:"name"`    //仓库名称
	Number  string  `json:"number"`  //仓库编号
	Type    string  `json:"type"`    //仓库类型
	Area    float64 `json:"area"`    //仓库面积
	City    string  `json:"city"`    //所在城市
	Address string  `json:"address"` //地址
	Manager string  `json:"manager"` //负责人
	Contact string  `json:"contact"` //联系方式
}
