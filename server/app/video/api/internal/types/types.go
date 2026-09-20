package types

// HealthData 与本域健康检查响应体一致，字段与统一响应中的 data 对应。
type HealthData struct {
	Service      string          `json:"service"`
	Dependencies map[string]bool `json:"dependencies"`
}
