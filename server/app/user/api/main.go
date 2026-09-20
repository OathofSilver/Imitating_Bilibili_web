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

	svcCtx := svc.NewServiceContext(c.Domain)
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/v1/user/health",
		Handler: middleware.CORS(handler.HealthHandler(svcCtx)),
	})

	fmt.Printf("user-api (用户域) listening at %s:%d\n", c.Host, c.Port)
	server.Start()
}
