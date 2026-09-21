package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisConf 是建立 Redis 连接所需的参数，五个域共用同一份定义。
//
// 默认值对应本机开发环境；口令不设默认值，必须由环境变量 REDIS_PASSWORD 提供。
type RedisConf struct {
	Addr     string `json:",default=127.0.0.1:6379,env=REDIS_ADDR"`
	Password string `json:",optional,env=REDIS_PASSWORD"`
	DB       int    `json:",default=0,env=REDIS_DB"`
}

// NewRedisClient 创建 Redis 客户端：只校验参数，不校验连通性。
//
// 服务启动使用它——中间件暂时不可用不应阻止进程起不来，
// 且保留客户端后，health 才能在 Redis 恢复后如实反映为可用。
func NewRedisClient(conf RedisConf) (*redis.Client, error) {
	if conf.Addr == "" {
		return nil, errors.New("store: redis 地址为空")
	}

	return redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Password: conf.Password,
		DB:       conf.DB,
	}), nil
}

// NewRedis 创建 Redis 客户端并立即校验连通性。
func NewRedis(conf RedisConf) (*redis.Client, error) {
	client, err := NewRedisClient(conf)
	if err != nil {
		return nil, err
	}

	if err := PingRedis(context.Background(), client); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

// PingRedis 校验 Redis 是否可连通，供健康检查与启动自检复用。
func PingRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return errors.New("store: redis 客户端为空")
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("store: redis 连通性校验失败: %w", err)
	}

	return nil
}

// CloseRedis 释放 Redis 客户端。
func CloseRedis(client *redis.Client) error {
	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("store: 关闭 redis 连接失败: %w", err)
	}

	return nil
}
