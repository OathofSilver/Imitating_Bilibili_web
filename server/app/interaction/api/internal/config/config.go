package config

import (
	"bilibili-web/server/common/auth"
	"bilibili-web/server/common/store"

	"github.com/zeromicro/go-zero/rest"
)

// Config 是 interaction 域服务的配置。
//
// 存储与鉴权配置的类型定义在 common/ 下由五个域复用：五域都需要 Redis
// （登录态撤销校验）与 JWT 密钥（本地验签），MySQL 目前仅用户域使用。
type Config struct {
	rest.RestConf
	Domain string `json:",default=interaction"`

	Mysql store.MysqlConf `json:",optional"`
	Redis store.RedisConf
	Auth  auth.AuthConf
}
