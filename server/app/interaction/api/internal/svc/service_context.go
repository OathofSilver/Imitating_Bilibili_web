package svc

import (
	"bilibili-web/server/app/interaction/api/internal/config"
	"bilibili-web/server/common/health"
	"bilibili-web/server/common/store"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceContext 持有本域服务运行期依赖。
type ServiceContext struct {
	Config config.Config
	Domain string

	// Redis 五域都要：凭证校验需查会话版本号，撤销后任意受保护接口都必须拒绝。
	Redis *redis.Client
}

// NewServiceContext 建立本域所需的连接。
//
// 刻意使用不校验连通性的构造方式：中间件暂时不可用不应阻止服务启动，
// 连通状态交由 health 接口实时反映（骨架决策 5）。
func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{Config: c, Domain: c.Domain}
	if svcCtx.Domain == "" {
		svcCtx.Domain = "interaction"
	}

	redisClient, err := store.NewRedisClient(c.Redis)
	if err != nil {
		logx.Errorf("[interaction] 创建 redis 客户端失败: %v", err)
	}
	svcCtx.Redis = redisClient

	return svcCtx
}

// Connections 暴露 health 检查所需的真实连接。
func (s *ServiceContext) Connections() health.Deps {
	return health.Deps{Redis: s.Redis}
}

// Close 释放本域持有的连接，应在进程退出时调用。
func (s *ServiceContext) Close() {
	if err := store.CloseRedis(s.Redis); err != nil {
		logx.Errorf("[interaction] 关闭 redis 连接失败: %v", err)
	}
}
