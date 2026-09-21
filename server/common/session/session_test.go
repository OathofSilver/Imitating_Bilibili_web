package session

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// testTTLSeconds 与 user 域配置的默认凭证有效期一致（7 天）。
const testTTLSeconds = 7 * 24 * 3600

// newTestClient 建连测试用 Redis；未设置 BW_TEST_REDIS_ADDR 时返回 nil，相关用例跳过。
//
// 地址由环境变量提供，测试代码内不出现连接串字面量（任务 1.3 的验收要求）。
func newTestClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("BW_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("跳过：未设置 BW_TEST_REDIS_ADDR，无法连接 Redis")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("跳过：Redis 不可用 (%v)", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

// uniqueUID 为一个用例产出独立的用户 ID，避免用例间互相污染版本号。
func uniqueUID() int64 {
	return -time.Now().UnixNano()
}

// 任务 2.4 验收：递增后键存在且 TTL 大于 0，键名符合前缀 + 结构版本规范。
func TestBumpSetsKeyWithTTL(t *testing.T) {
	client := newTestClient(t)
	store := NewStore(client, testTTLSeconds)
	ctx := context.Background()

	uid := uniqueUID()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), Key(uid)).Err()
	})

	ver, err := store.Bump(ctx, uid)
	if err != nil {
		t.Fatalf("递增会话版本失败: %v", err)
	}
	if ver != 1 {
		t.Fatalf("首次递增期望 1，实际 %d", ver)
	}

	ttl, err := client.TTL(ctx, Key(uid)).Result()
	if err != nil {
		t.Fatalf("读取 TTL 失败: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL 必须大于 0，实际 %v", ttl)
	}
	if want := time.Duration(testTTLSeconds*ttlFactor) * time.Second; ttl > want {
		t.Fatalf("TTL 不应超过 %v，实际 %v", want, ttl)
	}

	if got := Key(uid); got == "" {
		t.Fatalf("键名不应为空")
	}
	const wantPrefix = "bw:v1:auth:ver:"
	if got := Key(uid); len(got) <= len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("键名必须以 %q 开头，实际 %q", wantPrefix, got)
	}
}

// 未登录过的用户版本号为 0，且不产生错误。
func TestCurrentDefaultsToZeroWhenAbsent(t *testing.T) {
	client := newTestClient(t)
	store := NewStore(client, testTTLSeconds)

	ver, err := store.Current(context.Background(), uniqueUID())
	if err != nil {
		t.Fatalf("读取不存在的会话版本应返回 0 与 nil 错误，实际: %v", err)
	}
	if ver != 0 {
		t.Fatalf("不存在的会话版本期望 0，实际 %d", ver)
	}
}

// 连续递增必须单调增长，这是撤销语义成立的前提。
func TestBumpIsMonotonic(t *testing.T) {
	client := newTestClient(t)
	store := NewStore(client, testTTLSeconds)
	ctx := context.Background()

	uid := uniqueUID()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), Key(uid)).Err()
	})

	prev := int64(0)
	for i := int64(1); i <= 3; i++ {
		got, err := store.Bump(ctx, uid)
		if err != nil {
			t.Fatalf("第 %d 次递增失败: %v", i, err)
		}
		if got != i {
			t.Fatalf("第 %d 次递增期望 %d，实际 %d", i, i, got)
		}
		if got <= prev {
			t.Fatalf("版本号必须递增，前值 %d 现值 %d", prev, got)
		}
		prev = got
	}
}

// Matches 的核心语义：递增一次后旧版本立即不匹配。
func TestMatchesDetectsRevocation(t *testing.T) {
	client := newTestClient(t)
	store := NewStore(client, testTTLSeconds)
	ctx := context.Background()

	uid := uniqueUID()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), Key(uid)).Err()
	})

	issued, err := store.Bump(ctx, uid)
	if err != nil {
		t.Fatalf("递增会话版本失败: %v", err)
	}

	ok, err := store.Matches(ctx, uid, issued)
	if err != nil {
		t.Fatalf("校验会话版本失败: %v", err)
	}
	if !ok {
		t.Fatalf("刚签发的版本应当匹配")
	}

	// 模拟退出登录。
	if _, err := store.Bump(ctx, uid); err != nil {
		t.Fatalf("登出递增失败: %v", err)
	}

	ok, err = store.Matches(ctx, uid, issued)
	if err != nil {
		t.Fatalf("校验会话版本失败: %v", err)
	}
	if ok {
		t.Fatalf("递增后旧版本必须不再匹配")
	}
}

// client 为 nil（Redis 未就绪）时必须返回 ErrUnavailable，而不是 panic。
func TestNilClientReportsUnavailable(t *testing.T) {
	store := NewStore(nil, testTTLSeconds)
	ctx := context.Background()

	if _, err := store.Current(ctx, 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Current 期望 ErrUnavailable，实际: %v", err)
	}
	if _, err := store.Bump(ctx, 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Bump 期望 ErrUnavailable，实际: %v", err)
	}
	if _, err := store.Matches(ctx, 1, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Matches 期望 ErrUnavailable，实际: %v", err)
	}
}

// TTL 非正数时必须回退为一个可用的正值，避免产生永不过期的键。
func TestNewStoreFallsBackToPositiveTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
	}{
		{"有效期为零", 0},
		{"有效期为负", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(nil, tt.ttl)
			if store.ttl <= 0 {
				t.Fatalf("TTL 必须为正数，实际 %d", store.ttl)
			}
		})
	}
}

// 键被外部写入非法值时按 fail-closed 处理：返回错误而非静默当作 0。
func TestCurrentRejectsCorruptedValue(t *testing.T) {
	client := newTestClient(t)
	store := NewStore(client, testTTLSeconds)
	ctx := context.Background()

	uid := uniqueUID()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), Key(uid)).Err()
	})

	if err := client.Set(ctx, Key(uid), "not-a-number", time.Minute).Err(); err != nil {
		t.Fatalf("写入非法值失败: %v", err)
	}

	if _, err := store.Current(ctx, uid); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("非法值期望 ErrUnavailable，实际: %v", err)
	}
}
