package health

import (
	"context"
	"net"
	"os"
	"time"

	"bilibili-web/server/common/store"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Data 是健康检查的统一返回体，依赖不可用时仅标注状态，不返回业务错误码。
type Data struct {
	Service      string          `json:"service"`
	Dependencies map[string]bool `json:"dependencies"`
}

// Deps 是健康检查可用的真实连接。
//
// 为 nil 的项会回退为 TCP 端口探测，用于尚未接入客户端能力的中间件；
// 因此本结构可以只填该域实际持有的连接。
type Deps struct {
	Mysql sqlx.SqlConn
	Redis *redis.Client
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func probe(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkMysql(conn sqlx.SqlConn) bool {
	if conn == nil {
		// 本域未接入 MySQL，退回端口探测。
		return probe(envOr("MYSQL_ADDR", "127.0.0.1:3306"))
	}

	return store.PingMysql(context.Background(), conn) == nil
}

func checkRedis(client *redis.Client) bool {
	if client == nil {
		return probe(envOr("REDIS_ADDR", "127.0.0.1:6379"))
	}

	return store.PingRedis(context.Background(), client) == nil
}

// Dependencies 逐项给出依赖的连通状态。
//
// mysql 与 redis 在拿到真实连接时走连接级校验；rabbitmq 与 elasticsearch
// 尚未接入客户端，仍为端口探测。
func Dependencies(deps Deps) map[string]bool {
	return map[string]bool{
		"mysql":         checkMysql(deps.Mysql),
		"redis":         checkRedis(deps.Redis),
		"rabbitmq":      probe(envOr("RABBITMQ_ADDR", "127.0.0.1:5672")),
		"elasticsearch": probe(envOr("ES_ADDR", "127.0.0.1:9200")),
	}
}

// Check 汇总服务名与依赖连通状态。
func Check(service string, deps Deps) Data {
	return Data{Service: service, Dependencies: Dependencies(deps)}
}
