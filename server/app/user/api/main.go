package main

import (
	"flag"
	"fmt"
	"net/http"

	"bilibili-web/server/app/user/api/internal/config"
	"bilibili-web/server/app/user/api/internal/handler"
	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/common/middleware"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/user-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c)
	defer svcCtx.Close()

	// 受保护路由统一经 protected 包装，它内部决定是走真实鉴权还是
	// 因密钥缺失而整体拒绝（见 svc.ServiceContext.Protected）。
	protected := func(h http.HandlerFunc) http.HandlerFunc {
		return middleware.CORS(svcCtx.Protected(h))
	}
	open := func(h http.HandlerFunc) http.HandlerFunc {
		return middleware.CORS(h)
	}

	// 受保护接口：目标恒为凭证身份，不接受请求参数指定用户。
	protectedRoutes := []rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/user/logout",
			Handler: protected(handler.LogoutHandler(svcCtx)),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/user/me",
			Handler: protected(handler.MeHandler(svcCtx)),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/v1/user/me",
			Handler: protected(handler.UpdateMeHandler(svcCtx)),
		},
	}

	// 公开接口：无需登录。
	openRoutes := []rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/user/health",
			Handler: open(handler.HealthHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/user/register",
			Handler: open(handler.RegisterHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/user/login",
			Handler: open(handler.LoginHandler(svcCtx)),
		},
		{
			// 与 /me 前缀完全分离，避免「同前缀 + 可选路径参数」的匹配歧义
			// （design 决策 5）。
			Method:  http.MethodGet,
			Path:    "/api/v1/user/users/:id",
			Handler: open(handler.PublicProfileHandler(svcCtx)),
		},
	}

	// 先注册固定路径再注册带参路径：go-zero 的路由树对两者都能正确处理，
	// 但固定路径优先注册可读性更好，也便于人工核对清单。
	for _, route := range append(protectedRoutes, openRoutes...) {
		server.AddRoute(route)
	}

	fmt.Printf("user-api (用户域) listening at %s:%d\n", c.Host, c.Port)
	server.Start()
}
