package main

import (
	"flag"
	"fmt"
	"net/http"

	"bilibili-web/server/app/interaction/api/internal/config"
	"bilibili-web/server/app/interaction/api/internal/handler"
	"bilibili-web/server/app/interaction/api/internal/svc"
	"bilibili-web/server/common/middleware"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/interaction-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c)
	defer svcCtx.Close()
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/v1/interaction/health",
		Handler: middleware.CORS(handler.HealthHandler(svcCtx)),
	})

	fmt.Printf("interaction-api (互动域（点赞/投币/收藏）) listening at %s:%d\n", c.Host, c.Port)
	server.Start()
}
