// Package session 提供登录态的会话版本号管理，用于服务端撤销凭证。
//
// 设计见 design.md 决策 2。核心思路：JWT 本身无状态、签发后无法收回，
// 因此额外在 Redis 里为每个用户维护一个「会话版本号」，凭证签发时把当时的
// 版本号写进载荷；校验时比较两者，不一致即视为已撤销。
//
// 这样做的收益：
//   - 退出登录只需把版本号 +1，该用户已签发的**全部**凭证立即失效，
//     无需按设备或按 jti 逐个登记，键数量与用户数同阶而非与登录次数同阶。
//   - 撤销判定只需一次单键 GET，成本可控。
//
// 键名遵循 data/platform 的「业务前缀 + 结构版本 + 业务段」规范：
//
//	bw:v1:auth:ver:<uid>
package session

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// keyPrefix 是会话版本号键的固定前缀，末段 uid 由调用方拼接。
	keyPrefix = "bw:v1:auth:ver:"

	// ttlFactor 决定键的过期倍率。
	//
	// 取 2 倍凭证有效期：键自然过期时，该用户不可能还持有未过期凭证，
	// 语义自洽（design 决策 2）。若键在凭证仍有效时消失，读取会得到 0，
	// 与当前版本不匹配则拒绝——fail-closed，宁可让用户重新登录。
	ttlFactor = 2
)

// ErrUnavailable 表示 Redis 不可用，无法判定会话版本。
//
// 刻意与「版本不匹配」区分开：前者是依赖故障（映射为 50001），
// 后者是凭证已撤销（映射为 40100）。两者对客户端的处理方式完全不同。
var ErrUnavailable = errors.New("session: 会话存储不可用")

// Store 负责读写会话版本号。
type Store struct {
	client *redis.Client
	ttl    int
}

// NewStore 创建会话存储。
//
// tokenTTLSeconds 是凭证的有效期（秒），键的 TTL 取其 ttlFactor 倍。
// client 为 nil 时所有方法都会返回 ErrUnavailable，而不是 panic——
// 服务启动时 Redis 可能尚未就绪，此时应当让受保护接口以 50001 拒绝，
// 而不是让进程崩掉（design 决策 3 与 Risks）。
func NewStore(client *redis.Client, tokenTTLSeconds int) *Store {
	ttl := tokenTTLSeconds * ttlFactor
	if ttl <= 0 {
		ttl = ttlFactor
	}

	return &Store{client: client, ttl: ttl}
}

// Key 返回指定用户的会话版本号键名，便于日志与运维排查。
func Key(userID int64) string {
	return keyPrefix + strconv.FormatInt(userID, 10)
}

// Current 读取用户当前的会话版本号。
//
// 键不存在时返回 0（而非错误）：从未登录过的用户与版本号被重置的用户
// 语义相同，都视为版本 0。
func (s *Store) Current(ctx context.Context, userID int64) (int64, error) {
	if s == nil || s.client == nil {
		return 0, ErrUnavailable
	}

	raw, err := s.client.Get(ctx, Key(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%w: 读取会话版本失败: %v", ErrUnavailable, err)
	}

	ver, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// 值非法说明该键被外部污染，按版本 0 处理会让所有凭证失效（fail-closed）。
		return 0, fmt.Errorf("%w: 会话版本值非法 %q", ErrUnavailable, raw)
	}

	return ver, nil
}

// Bump 递增用户会话版本号并返回递增后的值，键的 TTL 同步重置。
//
// 登录与退出登录都调用它：
//   - 登录：新版本号写进新签发的凭证，使旧凭证全部失效（单点登录语义的替代）
//   - 退出：版本号再 +1，使刚签发的凭证也立即失效
func (s *Store) Bump(ctx context.Context, userID int64) (int64, error) {
	if s == nil || s.client == nil {
		return 0, ErrUnavailable
	}

	// pipeline 保证 INCR 与 EXPIRE 一次往返完成，避免 INCR 成功后进程中断
	// 留下一个永不过期的键。
	pipe := s.client.TxPipeline()
	incr := pipe.Incr(ctx, Key(userID))
	pipe.Expire(ctx, Key(userID), s.ttlDuration())

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("%w: 递增会话版本失败: %v", ErrUnavailable, err)
	}

	return incr.Val(), nil
}

// Matches 判断凭证携带的版本号是否与当前版本一致。
func (s *Store) Matches(ctx context.Context, userID, ver int64) (bool, error) {
	current, err := s.Current(ctx, userID)
	if err != nil {
		return false, err
	}

	return current == ver, nil
}

// ttlDuration 把秒级 TTL 转为 time.Duration。
func (s *Store) ttlDuration() time.Duration {
	return time.Duration(s.ttl) * time.Second
}
