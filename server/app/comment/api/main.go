package main

import (
	"flag"
	"fmt"
	"net/http"

	"bilibili-web/server/app/comment/api/internal/config"
	"bilibili-web/server/app/comment/api/internal/handler"
	"bilibili-web/server/app/comment/api/internal/svc"
	"bilibili-web/server/common/middleware"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/comment-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c.Domain)
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/v1/comment/health",
		Handler: middleware.CORS(handler.HealthHandler(svcCtx)),
	})

	fmt.Printf("comment-api (评论域) listening at %s:%d\n", c.Host, c.Port)
	server.Start()
}
