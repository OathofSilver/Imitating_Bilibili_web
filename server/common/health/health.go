package health

import (
	"net"
	"os"
	"time"
)

// Data 是健康检查的统一返回体，依赖不可用时仅标注状态，不返回业务错误码。
type Data struct {
	Service      string          `json:"service"`
	Dependencies map[string]bool `json:"dependencies"`
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

// Dependencies 探测骨架依赖的中间件的连通性。
func Dependencies() map[string]bool {
	return map[string]bool{
		"mysql":         probe(envOr("MYSQL_ADDR", "127.0.0.1:3306")),
		"redis":         probe(envOr("REDIS_ADDR", "127.0.0.1:6379")),
		"rabbitmq":      probe(envOr("RABBITMQ_ADDR", "127.0.0.1:5672")),
		"elasticsearch": probe(envOr("ES_ADDR", "127.0.0.1:9200")),
	}
}

func Check(service string) Data {
	return Data{Service: service, Dependencies: Dependencies()}
}
