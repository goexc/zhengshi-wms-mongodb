package main

import (
	"api/internal/config"
	"api/internal/handler"
	"api/internal/svc"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"net/http"
	"strings"
)

var configFile = flag.String("f", "etc/main.yaml", "the config file")

func main() {
	flag.Parse()

	logx.DisableStat()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	//server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(nil, func(w http.ResponseWriter) {
	//	w.Header().Set("Access-Control-Allow-Origin", "*")
	//	w.Header().Set("Access-Control-Allow-Headers", "*")
	//	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	//	w.Header().Set("Access-Control-Expose-Headers", "Authorization,DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range, Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
	//	w.Header().Set("Access-Control-Allow-Credentials", "true")
	//}, "*"))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	//全局中间件
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { //如果是跨域预检请求，那就别再继续执行。
			//业务逻辑
			//1.demo环境配置，防止增、删、改操作
			//switch r.Method {
			//case "GET", "OPTIONS":
			//case "POST":
			//	if r.URL.Path != "/auth/login" {
			//		w.Write([]byte(`{"code": 200, "msg":"成功"}`))
			//		return
			//	}
			//default:
			//	w.Write([]byte(`{"code": 200, "msg":"成功"}`))
			//	return
			//}

			//2.权限校验
			//2.0 api白名单
			whitelist := []string{"/auth/login", "/auth/register"}
			for _, white := range whitelist {
				if white == r.URL.Path {
					next(w, r)
					return
				}
			}
			//2.1 提取用户id
			if r.Context().Value("uid") == nil {
				fmt.Printf("[Error]Token为空\n")
				w.Write([]byte(`{"code": 401, "msg":"请登录"}`))
				return
			}
			uid := r.Context().Value("uid").(string)

			//2.3 权限校验
			//2.3.1 判断用户是否具有超级管理员角色
			var isSystemRole bool //超级管理员标记
			var casbinRoles []string
			casbinRoles, err := ctx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", uid))
			for _, casbinRole := range casbinRoles {
				if strings.TrimPrefix(casbinRole, "role_") == ctx.Config.Ids.Role {
					isSystemRole = true
				}
			}

			//超级管理员角色全部放行
			if isSystemRole {
				fmt.Println("超级管理员角色全部放行")
				next(w, r)
				return
			}

			//2.3.2 普通角色执行权限校验
			has, err := ctx.Enforcer.Enforce(fmt.Sprintf("user_%s", uid), r.URL.Path, r.Method)
			if err != nil {
				fmt.Printf("[Error]权限校验[user_%s][%s][%s]:%s\n", uid, r.Method, r.URL.Path, err.Error())
				w.Write([]byte(`{"code": 500, "msg":"服务内部错误"}`))
				return
			}
			fmt.Printf("[Debug]权限校验[%s][%s]:%v\n", r.Method, r.URL.Path, has)
			if !has {
				fmt.Printf("[Error][user_%s]权限校验[%s][%s]不通过\n", uid, r.Method, r.URL.Path)
				//http.StatusUnauthorized
				w.Write([]byte(`{"code": 403, "msg":"没有权限"}`))
				return
			}

			next(w, r)
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
