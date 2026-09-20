package logic

import (
	"bilibili-web/server/app/video/api/internal/svc"
	"bilibili-web/server/app/video/api/internal/types"
	"bilibili-web/server/common/health"
)

// Health 返回本域服务的健康状态，依赖不可用时仅标注 false，不返回错误。
func Health(svcCtx *svc.ServiceContext) types.HealthData {
	data := health.Check(svcCtx.Domain)
	return types.HealthData{Service: data.Service, Dependencies: data.Dependencies}
}
