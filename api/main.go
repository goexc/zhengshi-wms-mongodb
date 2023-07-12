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
)

var configFile = flag.String("f", "etc/main.yaml", "the config file")

func main() {
	flag.Parse()

	logx.DisableStat()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	/*
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
				uid := r.Context().Value("uid").(string)
				if uid == "" {
					fmt.Printf("[Error]Token为空\n")
					w.Write([]byte(`{"code": 401, "msg":"请登录"}`))
					return
				}

				//2.3 权限校验
				//has, err := ctx.Enforcer.Enforce(fmt.Sprintf("role_1"), fmt.Sprintf("menu_%d", menu.Id), "")
				has, err := ctx.Enforcer.Enforce(fmt.Sprintf("user_%d", uid), r.URL.Path, r.Method)
				if err != nil {
					fmt.Printf("[Error]权限校验[user_%d][%s][%s]:%s\n", uid, r.Method, r.URL.Path, err.Error())
					w.Write([]byte(`{"code": 500, "msg":"服务内部错误"}`))
					return
				}
				fmt.Printf("[Debug]权限校验[%s][%s]:%v\n", r.Method, r.URL.Path, has)
				if !has {
					fmt.Printf("[Error][user_%d]权限校验[%s][%s]不通过\n", uid, r.Method, r.URL.Path)
					//http.StatusUnauthorized
					w.Write([]byte(`{"code": 403, "msg":"没有权限"}`))
					return
				}

				next(w, r)
			}
		})
	*/

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
