package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"bilibili-web/server/common/snowflake"
	"bilibili-web/server/common/store"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// go-zero 的 sqlx 按「驱动 + 数据源」全局缓存连接池，因此整个测试包只建立一次连接，
// 且不在此处关闭——中途 Close 会让后续测试拿到已关闭的连接池。
var (
	testConnOnce sync.Once
	testConn     sqlx.SqlConn
	testConnErr  error
)

// newTestConn 返回测试用连接。
//
// 连接串完全由环境变量提供，测试代码内不出现任何口令字面量（任务 1.3 的验收要求）。
// 未提供 BW_TEST_MYSQL_DSN 时跳过而非失败，使未启动中间件时 `go test ./...` 仍可整体通过。
// 本机执行方式：
//
//	BW_TEST_MYSQL_DSN='<账号>:<口令>@tcp(127.0.0.1:13306)/bilibili_web?charset=utf8mb4&parseTime=true&loc=Local' \
//	  go test ./app/user/model/...
func newTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()

	testConnOnce.Do(func() {
		dsn := os.Getenv("BW_TEST_MYSQL_DSN")
		if dsn == "" {
			return
		}

		testConn, testConnErr = store.NewMysql(sqlx.SqlConf{DataSource: dsn, DriverName: "mysql"})
	})

	if testConnErr != nil {
		t.Skipf("跳过：MySQL 不可用 (%v)", testConnErr)
	}
	if testConn == nil {
		t.Skip("跳过：未设置 BW_TEST_MYSQL_DSN，无法连接 MySQL")
	}

	return testConn
}

// insertUser 插入一个用户并登记清理逻辑，返回其 Snowflake ID。
func insertUser(t *testing.T, conn sqlx.SqlConn, m UsersModel) int64 {
	t.Helper()

	// 用户名带纳秒后缀，避免同一库上并发/重复执行时撞唯一索引。
	username := fmt.Sprintf("ut_%d", time.Now().UnixNano())

	id := snowflake.MustID()
	user := &Users{
		Id:           id,
		Username:     username,
		PasswordHash: "$2a$10$testhashnotreal",
		Nickname:     username,
	}
	if _, err := m.Insert(context.Background(), user); err != nil {
		t.Fatalf("插入测试用户失败: %v", err)
	}

	// 测试数据做物理清理，避免污染开发库。
	t.Cleanup(func() {
		_, _ = conn.ExecCtx(context.Background(), "delete from users where id = ?", id)
	})

	return id
}

// 场景一：按 ID 查询已软删除用户必须返回 ErrNotFound。
func TestFindOneIgnoresSoftDeletedUser(t *testing.T) {
	conn := newTestConn(t)
	m := NewUsersModel(conn)
	ctx := context.Background()

	id := insertUser(t, conn, m)

	if _, err := m.FindOne(ctx, id); err != nil {
		t.Fatalf("软删除前应能查到用户，实际错误: %v", err)
	}

	if err := m.Delete(ctx, id); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	if _, err := m.FindOne(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("软删除后 FindOne 期望 ErrNotFound，实际: %v", err)
	}
}

// 场景二：软删除后该行仍在库中，且 deleted_at 非空——即未被物理删除。
func TestDeleteIsSoftDelete(t *testing.T) {
	conn := newTestConn(t)
	m := NewUsersModel(conn)
	ctx := context.Background()

	id := insertUser(t, conn, m)

	if err := m.Delete(ctx, id); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	var total int64
	if err := conn.QueryRowCtx(ctx, &total, "select count(*) from users where id = ?", id); err != nil {
		t.Fatalf("统计行数失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("软删除后行应当仍在库中，实际行数: %d", total)
	}

	var marked int64
	if err := conn.QueryRowCtx(
		ctx, &marked, "select count(*) from users where id = ? and deleted_at is not null", id,
	); err != nil {
		t.Fatalf("统计软删除标记失败: %v", err)
	}
	if marked != 1 {
		t.Fatalf("软删除后 deleted_at 应非空，实际匹配行数: %d", marked)
	}
}

// 场景三：UpdateProfile 改昵称后 username 必须保持原值。
func TestUpdateProfileKeepsUsername(t *testing.T) {
	conn := newTestConn(t)
	m := NewUsersModel(conn)
	ctx := context.Background()

	id := insertUser(t, conn, m)

	before, err := m.FindOne(ctx, id)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	const newNickname = "新昵称"
	if err := m.UpdateProfile(ctx, id, ProfileUpdate{
		Nickname:  newNickname,
		AvatarUrl: "https://example.com/a.png",
		Signature: "签名",
		Gender:    1,
		Birthday:  sql.NullTime{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatalf("更新资料失败: %v", err)
	}

	after, err := m.FindOne(ctx, id)
	if err != nil {
		t.Fatalf("更新后查询用户失败: %v", err)
	}

	if after.Nickname != newNickname {
		t.Fatalf("昵称未更新，期望 %q 实际 %q", newNickname, after.Nickname)
	}
	if after.Username != before.Username {
		t.Fatalf("username 被意外修改，原值 %q 现值 %q", before.Username, after.Username)
	}
	if after.PasswordHash != before.PasswordHash {
		t.Fatalf("password_hash 被资料更新意外改动")
	}
}

// 按用户名查询同样必须跳过已软删除的用户。
func TestFindOneByUsernameIgnoresSoftDeletedUser(t *testing.T) {
	conn := newTestConn(t)
	m := NewUsersModel(conn)
	ctx := context.Background()

	id := insertUser(t, conn, m)

	before, err := m.FindOne(ctx, id)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	if _, err := m.FindOneByUsername(ctx, before.Username); err != nil {
		t.Fatalf("软删除前按用户名应能查到，实际错误: %v", err)
	}

	if err := m.Delete(ctx, id); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	if _, err := m.FindOneByUsername(ctx, before.Username); !errors.Is(err, ErrNotFound) {
		t.Fatalf("软删除后 FindOneByUsername 期望 ErrNotFound，实际: %v", err)
	}
}
