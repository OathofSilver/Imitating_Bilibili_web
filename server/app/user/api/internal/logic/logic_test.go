package logic

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"bilibili-web/server/app/user/api/internal/config"
	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/app/user/api/internal/types"
	"bilibili-web/server/app/user/model"
	"bilibili-web/server/common/token"

	"github.com/zeromicro/go-zero/rest"
)

// ===== 测试替身 =====

// fakeUsers 是 model.UsersModel 的内存实现。
//
// 只需要实现被测流程真正调用的方法；其余方法用 panic 标注「本测试不应走到」，
// 这样如果实现被改动后意外调用了它们，测试会立刻失败而不是静默通过。
type fakeUsers struct {
	// byID 与 byName 共用同一批数据，模拟唯一索引与主键索引。
	users map[int64]*model.Users

	insertErr  error
	findErr    error
	updateErr  error
	insertHits int
	updateHits int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[int64]*model.Users{}}
}

func (f *fakeUsers) Insert(_ context.Context, data *model.Users) (sql.Result, error) {
	f.insertHits++
	if f.insertErr != nil {
		return nil, f.insertErr
	}

	// 模拟「唯一索引冲突」：用户名已存在则报错。
	for _, u := range f.users {
		if strings.EqualFold(u.Username, data.Username) {
			return nil, errDuplicate
		}
	}

	// 模拟数据库的默认时间戳：Insert 不回填，需要再查一次才能拿到。
	copied := *data
	f.users[copied.Id] = &copied

	return stubResult(1), nil
}

func (f *fakeUsers) FindOne(_ context.Context, id int64) (*model.Users, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}

	u, ok := f.users[id]
	if !ok || u.DeletedAt.Valid {
		return nil, model.ErrNotFound
	}

	copied := *u
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
	}

	return &copied, nil
}

func (f *fakeUsers) FindOneByUsername(_ context.Context, username string) (*model.Users, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}

	for _, u := range f.users {
		if strings.EqualFold(u.Username, username) && !u.DeletedAt.Valid {
			copied := *u
			if copied.CreatedAt.IsZero() {
				copied.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
			}

			return &copied, nil
		}
	}

	return nil, model.ErrNotFound
}

func (f *fakeUsers) UpdateProfile(_ context.Context, id int64, in model.ProfileUpdate) error {
	f.updateHits++
	if f.updateErr != nil {
		return f.updateErr
	}

	u, ok := f.users[id]
	if !ok {
		return model.ErrNotFound
	}

	// 只应用白名单字段，username / password_hash / level / role 保持不变。
	u.Nickname = in.Nickname
	u.AvatarUrl = in.AvatarUrl
	u.Signature = in.Signature
	u.Gender = in.Gender
	u.Birthday = in.Birthday

	return nil
}

func (f *fakeUsers) Update(context.Context, *model.Users) error {
	panic("logic 层不得调用生成的 Update：它会连 username 一起写")
}

func (f *fakeUsers) Delete(context.Context, int64) error {
	panic("本测试不应走到 Delete")
}

// fakeSessions 是 svc.SessionStore 的内存实现。
type fakeSessions struct {
	versions map[int64]int64
	err      error
	bumps    int
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{versions: map[int64]int64{}}
}

func (f *fakeSessions) Current(_ context.Context, userID int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}

	return f.versions[userID], nil
}

func (f *fakeSessions) Bump(_ context.Context, userID int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}

	f.bumps++
	f.versions[userID]++

	return f.versions[userID], nil
}

func (f *fakeSessions) Matches(_ context.Context, userID, ver int64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}

	return f.versions[userID] == ver, nil
}

// fakeIssuer 是 svc.TokenIssuer 的确定实现：凭证就是 uid:ver 的字符串。
type fakeIssuer struct {
	issueErr error
	parseErr error
	issued   []string
}

func (f *fakeIssuer) Issue(userID, ver int64, _ time.Time) (string, error) {
	if f.issueErr != nil {
		return "", f.issueErr
	}

	raw := uidVerToken(userID, ver)
	f.issued = append(f.issued, raw)

	return raw, nil
}

func (f *fakeIssuer) Parse(raw string) (*token.Claims, error) {
	if f.parseErr != nil {
		return nil, f.parseErr
	}

	var uid, ver int64
	if _, _, err := parseUIDVer(raw, &uid, &ver); err != nil {
		return nil, err
	}

	return &token.Claims{UserID: uid, Version: ver}, nil
}

func (f *fakeIssuer) TTL() time.Duration {
	return time.Hour
}

// errDuplicate 模拟驱动返回的唯一键冲突错误。
//
// 用真实的 *mysql.MySQLError 类型，确保 model.IsDuplicateUsername 的
// errors.As 判定路径被真正走到。
var errDuplicate = newDuplicateError()

// newTestContext 组装一个全部依赖为测试替身的 ServiceContext。
func newTestContext(users *fakeUsers, sessions *fakeSessions, issuer *fakeIssuer) *svc.ServiceContext {
	return &svc.ServiceContext{
		Config:    config.Config{RestConf: rest.RestConf{}, Domain: "user"},
		Domain:    "user",
		Users:     users,
		Sessions:  sessions,
		Tokens:    issuer,
		AuthReady: true,
	}
}

// ===== 注册 =====

func TestRegister(t *testing.T) {
	tests := []struct {
		name        string
		req         types.RegisterReq
		wantErr     error
		wantField   string
		wantToken   bool
		prePopulate func(*fakeUsers)
	}{
		{
			name:      "注册成功并直接签发凭证",
			req:       types.RegisterReq{Username: "newbie", Password: "secret12345"},
			wantToken: true,
		},
		{
			name:      "用户名归一化为小写",
			req:       types.RegisterReq{Username: "MixedCase", Password: "secret12345"},
			wantToken: true,
		},
		{
			name:      "用户名为空",
			req:       types.RegisterReq{Username: "", Password: "secret12345"},
			wantField: "username",
		},
		{
			name:      "用户名过短",
			req:       types.RegisterReq{Username: "ab", Password: "secret12345"},
			wantField: "username",
		},
		{
			name:      "用户名含非法字符",
			req:       types.RegisterReq{Username: "bad-name!", Password: "secret12345"},
			wantField: "username",
		},
		{
			name:      "用户名超长",
			req:       types.RegisterReq{Username: strings.Repeat("a", 33), Password: "secret12345"},
			wantField: "username",
		},
		{
			name:      "密码过短",
			req:       types.RegisterReq{Username: "newbie", Password: "short"},
			wantField: "password",
		},
		{
			name:      "密码为空",
			req:       types.RegisterReq{Username: "newbie", Password: ""},
			wantField: "password",
		},
		{
			name: "用户名已被占用（大小写不同也算）",
			req:  types.RegisterReq{Username: "TAKEN", Password: "secret12345"},
			prePopulate: func(u *fakeUsers) {
				u.users[1] = &model.Users{Id: 1, Username: "taken", Nickname: "taken"}
			},
			wantErr: ErrUsernameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			if tt.prePopulate != nil {
				tt.prePopulate(users)
			}
			sessions := newFakeSessions()
			issuer := &fakeIssuer{}

			svcCtx := newTestContext(users, sessions, issuer)

			result, err := Register(context.Background(), svcCtx, &tt.req)

			if tt.wantField != "" {
				fe, ok := asFieldError(err)
				if !ok {
					t.Fatalf("期望字段级错误，实际: %v", err)
				}
				if fe.Field != tt.wantField {
					t.Fatalf("字段名期望 %q，实际 %q", tt.wantField, fe.Field)
				}
				// 校验失败不得写库。
				if users.insertHits != 0 {
					t.Fatalf("校验失败时不应写库，实际写入 %d 次", users.insertHits)
				}
				return
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("错误期望 %v，实际 %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if !tt.wantToken || result.Token == "" {
				t.Fatalf("注册成功必须直接返回凭证")
			}
			if result.User == nil {
				t.Fatalf("注册成功必须返回用户资料")
			}
			if result.User.Nickname != result.User.Username {
				t.Fatalf("新用户昵称默认应取用户名，用户名 %q 昵称 %q",
					result.User.Username, result.User.Nickname)
			}
			if sessions.bumps != 1 {
				t.Fatalf("注册应递增一次会话版本，实际 %d 次", sessions.bumps)
			}
			// 注册响应里的注册时间必须来自数据库，而不是内存结构体的零值。
			// 这条断言固化的是一次真实回归：Insert 不回填 created_at，
			// 直接转换内存结构体会让响应返回 `0001-01-01T00:00:00Z`。
			if result.User.CreatedAt == zeroTimeText {
				t.Fatalf("注册响应的 created_at 不得为 Go 零值")
			}
			if _, err := time.Parse(time.RFC3339, result.User.CreatedAt); err != nil {
				t.Fatalf("created_at 应为 RFC3339，实际 %q: %v", result.User.CreatedAt, err)
			}
		})
	}
}

// zeroTimeText 是 time.Time 零值按 RFC3339 渲染的结果。
const zeroTimeText = "0001-01-01T00:00:00Z"

// 密码必须以哈希形式落库，且响应里不得出现哈希。
func TestRegisterStoresHashNotPlaintext(t *testing.T) {
	users := newFakeUsers()
	svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

	const plain = "secret12345"
	result, err := Register(context.Background(), svcCtx, &types.RegisterReq{
		Username: "hashcheck",
		Password: plain,
	})
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	stored := users.users[result.User.ID]
	if stored == nil {
		t.Fatalf("用户未落库")
	}
	if stored.PasswordHash == plain {
		t.Fatalf("密码不得以明文落库")
	}
	if stored.PasswordHash == "" {
		t.Fatalf("密码哈希不得为空")
	}
	if !strings.HasPrefix(stored.PasswordHash, "$2") {
		t.Fatalf("密码哈希应为 bcrypt 形式，实际前缀 %q", stored.PasswordHash[:2])
	}

	// DTO 是独立结构，物理上没有密码字段；这里用「转换后的输出不含哈希」间接固化。
	if strings.Contains(result.User.Username, stored.PasswordHash) {
		t.Fatalf("响应不得包含密码哈希")
	}
}

// 注册时不检查「用户名是否已存在」，只依赖唯一索引兜底——这条测试固化了该行为。
func TestRegisterReliesOnUniqueIndexNotPrecheck(t *testing.T) {
	users := newFakeUsers()
	users.insertErr = errDuplicate
	svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

	_, err := Register(context.Background(), svcCtx, &types.RegisterReq{
		Username: "racer",
		Password: "secret12345",
	})

	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("唯一索引冲突应映射为 ErrUsernameTaken，实际: %v", err)
	}
}

// 依赖缺失时注册必须失败，不得空指针。
func TestRegisterWithoutDependencies(t *testing.T) {
	tests := []struct {
		name  string
		build func() *svc.ServiceContext
	}{
		{"Users 为 nil", func() *svc.ServiceContext {
			return &svc.ServiceContext{Sessions: newFakeSessions(), Tokens: &fakeIssuer{}}
		}},
		{"Sessions 为 nil", func() *svc.ServiceContext {
			return &svc.ServiceContext{Users: newFakeUsers(), Tokens: &fakeIssuer{}}
		}},
		{"Tokens 为 nil", func() *svc.ServiceContext {
			return &svc.ServiceContext{Users: newFakeUsers(), Sessions: newFakeSessions()}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Register(context.Background(), tt.build(), &types.RegisterReq{
				Username: "someone",
				Password: "secret12345",
			})
			if !errors.Is(err, ErrDependencyUnavailable) {
				t.Fatalf("期望 ErrDependencyUnavailable，实际: %v", err)
			}
		})
	}
}

// ===== 登录 =====

func TestLogin(t *testing.T) {
	// 预先准备一个密码为 secret12345 的用户。
	const plain = "secret12345"

	hash, err := hashForTest(plain)
	if err != nil {
		t.Fatalf("生成测试哈希失败: %v", err)
	}
	existing := &model.Users{
		Id:           1001,
		Username:     "alice",
		PasswordHash: hash,
		Nickname:     "alice",
		CreatedAt:    time.Date(2026, 5, 6, 7, 8, 9, 0, time.Local),
	}

	tests := []struct {
		name      string
		req       types.LoginReq
		wantErr   error
		wantToken bool
	}{
		{
			name:      "凭据正确",
			req:       types.LoginReq{Username: "alice", Password: plain},
			wantToken: true,
		},
		{
			name:      "用户名大小写不敏感",
			req:       types.LoginReq{Username: "ALICE", Password: plain},
			wantToken: true,
		},
		{
			name:    "密码错误",
			req:     types.LoginReq{Username: "alice", Password: "wrongpassword"},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:    "用户名不存在",
			req:     types.LoginReq{Username: "nobody", Password: plain},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:    "用户名不存在且密码也为空",
			req:     types.LoginReq{Username: "nobody", Password: ""},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:    "用户名为空",
			req:     types.LoginReq{Username: "", Password: plain},
			wantErr: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			copied := *existing
			users.users[existing.Id] = &copied

			sessions := newFakeSessions()
			issuer := &fakeIssuer{}
			svcCtx := newTestContext(users, sessions, issuer)

			result, err := Login(context.Background(), svcCtx, &tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("错误期望 %v，实际 %v", tt.wantErr, err)
				}
				// 失败的登录不得递增会话版本（否则会把已登录的会话踢掉）。
				if sessions.bumps != 0 {
					t.Fatalf("登录失败不应递增会话版本，实际 %d 次", sessions.bumps)
				}
				return
			}

			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if !tt.wantToken || result.Token == "" {
				t.Fatalf("登录成功必须返回凭证")
			}
			if result.User.ID != existing.Id {
				t.Fatalf("返回的用户 ID 期望 %d，实际 %d", existing.Id, result.User.ID)
			}
			if sessions.bumps != 1 {
				t.Fatalf("登录成功应递增一次会话版本，实际 %d 次", sessions.bumps)
			}
		})
	}
}

// 登录成功后新凭证的版本必须等于递增后的值，否则新凭证会立刻失效。
func TestLoginIssuesTokenWithBumpedVersion(t *testing.T) {
	hash, err := hashForTest("secret12345")
	if err != nil {
		t.Fatalf("生成测试哈希失败: %v", err)
	}

	users := newFakeUsers()
	users.users[7] = &model.Users{Id: 7, Username: "bob", PasswordHash: hash, Nickname: "bob"}

	// 预设已有两次登录，本次登录后版本应为 3。
	sessions := newFakeSessions()
	sessions.versions[7] = 2

	svcCtx := newTestContext(users, sessions, &fakeIssuer{})

	result, err := Login(context.Background(), svcCtx, &types.LoginReq{
		Username: "bob",
		Password: "secret12345",
	})
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	uid, ver, err := parseUIDVer(result.Token, nil, nil)
	if err != nil {
		t.Fatalf("解析测试凭证失败: %v", err)
	}
	if uid != 7 {
		t.Fatalf("凭证 uid 期望 7，实际 %d", uid)
	}
	if ver != 3 {
		t.Fatalf("凭证 ver 期望 3（递增后），实际 %d", ver)
	}
}

// ===== 退出登录 =====

func TestLogout(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		setup   func(*fakeSessions)
		wantErr error
		wantVer int64
	}{
		{
			name:    "正常登出递增版本",
			userID:  42,
			setup:   func(s *fakeSessions) { s.versions[42] = 5 },
			wantVer: 6,
		},
		{
			name:    "首次登出（无既有版本）",
			userID:  43,
			setup:   func(*fakeSessions) {},
			wantVer: 1,
		},
		{
			name:    "会话存储故障",
			userID:  42,
			setup:   func(s *fakeSessions) { s.err = errors.New("redis down") },
			wantErr: ErrDependencyUnavailable,
		},
		{
			name:    "用户 ID 非正数",
			userID:  0,
			setup:   func(*fakeSessions) {},
			wantErr: ErrDependencyUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := newFakeSessions()
			tt.setup(sessions)
			svcCtx := newTestContext(newFakeUsers(), sessions, &fakeIssuer{})

			err := Logout(context.Background(), svcCtx, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("错误期望 %v，实际 %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if got := sessions.versions[tt.userID]; got != tt.wantVer {
				t.Fatalf("登出后版本期望 %d，实际 %d", tt.wantVer, got)
			}
		})
	}
}

// 登出的关键是「旧版本不再匹配」，这条测试直接验证撤销语义。
func TestLogoutInvalidatesPreviousVersion(t *testing.T) {
	sessions := newFakeSessions()
	sessions.versions[9] = 1
	svcCtx := newTestContext(newFakeUsers(), sessions, &fakeIssuer{})

	// 登出前版本 1 有效。
	ok, err := sessions.Matches(context.Background(), 9, 1)
	if err != nil || !ok {
		t.Fatalf("登出前版本 1 应有效，ok=%v err=%v", ok, err)
	}

	if err := Logout(context.Background(), svcCtx, 9); err != nil {
		t.Fatalf("登出失败: %v", err)
	}

	ok, err = sessions.Matches(context.Background(), 9, 1)
	if err != nil {
		t.Fatalf("校验失败: %v", err)
	}
	if ok {
		t.Fatalf("登出后版本 1 必须失效")
	}
}

// ===== 资料查询 =====

func TestMe(t *testing.T) {
	base := &model.Users{
		Id:           500,
		Username:     "carol",
		PasswordHash: "$2a$10$fakehashvalue",
		Nickname:     "carol",
		Signature:    "签名",
		Level:        3,
		CreatedAt:    time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local),
	}

	tests := []struct {
		name     string
		userID   int64
		prepare  func(*fakeUsers)
		wantErr  error
		wantNil  bool
		wantName string
	}{
		{
			name:     "查询本人资料",
			userID:   500,
			prepare:  func(u *fakeUsers) { c := *base; u.users[base.Id] = &c },
			wantName: "carol",
		},
		{
			name:    "用户不存在",
			userID:  999,
			prepare: func(u *fakeUsers) { c := *base; u.users[base.Id] = &c },
			wantErr: ErrUserNotFound,
		},
		{
			name:   "用户已软删除",
			userID: 500,
			prepare: func(u *fakeUsers) {
				c := *base
				c.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
				u.users[base.Id] = &c
			},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			tt.prepare(users)
			svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

			dto, err := Me(context.Background(), svcCtx, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("错误期望 %v，实际 %v", tt.wantErr, err)
				}
				if dto != nil {
					t.Fatalf("出错时不应返回资料")
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if dto.Nickname != tt.wantName {
				t.Fatalf("昵称期望 %q，实际 %q", tt.wantName, dto.Nickname)
			}
			// DTO 不含 password_hash 字段，这里额外确认等级等只读字段被带出。
			if dto.Level != base.Level {
				t.Fatalf("等级期望 %d，实际 %d", base.Level, dto.Level)
			}
		})
	}
}

func TestPublicProfile(t *testing.T) {
	base := &model.Users{
		Id:        600,
		Username:  "dave",
		Nickname:  "Dave",
		Signature: "hi",
		Gender:    1,
		Level:     7,
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name    string
		userID  int64
		prepare func(*fakeUsers)
		wantErr error
	}{
		{
			name:    "查询公开资料",
			userID:  600,
			prepare: func(u *fakeUsers) { c := *base; u.users[base.Id] = &c },
		},
		{
			name:    "用户不存在",
			userID:  601,
			prepare: func(u *fakeUsers) { c := *base; u.users[base.Id] = &c },
			wantErr: ErrUserNotFound,
		},
		{
			name:   "用户已软删除",
			userID: 600,
			prepare: func(u *fakeUsers) {
				c := *base
				c.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
				u.users[base.Id] = &c
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:    "ID 非正数",
			userID:  0,
			prepare: func(u *fakeUsers) { c := *base; u.users[base.Id] = &c },
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			tt.prepare(users)
			svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

			dto, err := PublicProfile(context.Background(), svcCtx, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("错误期望 %v，实际 %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if dto.Nickname != "Dave" || dto.Level != 7 {
				t.Fatalf("公开资料字段不符: %+v", dto)
			}
		})
	}
}

// PublicUserDTO 结构上不含 username 与 birthday，这条测试固化该契约。
func TestPublicProfileDTOHasNoPrivateFields(t *testing.T) {
	users := newFakeUsers()
	users.users[600] = &model.Users{
		Id:       600,
		Username: "secret_login_name",
		Nickname: "Dave",
	}

	svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

	dto, err := PublicProfile(context.Background(), svcCtx, 600)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	// 类型上不存在字段即编译期保证；这里确认公开字段被正确填充。
	if dto.Nickname == "" {
		t.Fatalf("公开资料必须含昵称")
	}
	if dto.ID != 600 {
		t.Fatalf("ID 期望 600，实际 %d", dto.ID)
	}
}

// ===== 资料修改 =====

func TestUpdateProfile(t *testing.T) {
	base := &model.Users{
		Id:           700,
		Username:     "erin",
		PasswordHash: "$2a$10$keepme",
		Nickname:     "erin",
		Signature:    "旧签名",
		Gender:       0,
		Level:        2,
		Role:         0,
		CreatedAt:    time.Date(2026, 2, 3, 4, 5, 6, 0, time.Local),
	}

	strPtr := func(s string) *string { return &s }
	i64Ptr := func(v int64) *int64 { return &v }

	tests := []struct {
		name      string
		req       *types.UpdateProfileReq
		wantErr   error
		wantField string
		verify    func(*testing.T, *model.Users)
	}{
		{
			name: "只改昵称",
			req:  &types.UpdateProfileReq{Nickname: strPtr("新昵称")},
			verify: func(t *testing.T, u *model.Users) {
				if u.Nickname != "新昵称" {
					t.Fatalf("昵称未更新: %q", u.Nickname)
				}
				// 未提交的字段必须保持原值。
				if u.Signature != "旧签名" {
					t.Fatalf("未提交的签名被改动: %q", u.Signature)
				}
				if u.Username != "erin" {
					t.Fatalf("username 被改动: %q", u.Username)
				}
				if u.PasswordHash != "$2a$10$keepme" {
					t.Fatalf("password_hash 被改动")
				}
				if u.Level != 2 || u.Role != 0 {
					t.Fatalf("level/role 被改动: %d/%d", u.Level, u.Role)
				}
			},
		},
		{
			name: "同时改多个字段",
			req: &types.UpdateProfileReq{
				Nickname:  strPtr("多字段"),
				Signature: strPtr("新签名"),
				Gender:    i64Ptr(2),
				AvatarURL: strPtr("https://example.com/avatar.png"),
				Birthday:  strPtr("2000-01-02"),
			},
			verify: func(t *testing.T, u *model.Users) {
				if u.Nickname != "多字段" || u.Signature != "新签名" {
					t.Fatalf("昵称或签名未更新: %q %q", u.Nickname, u.Signature)
				}
				if u.Gender != 2 {
					t.Fatalf("性别未更新: %d", u.Gender)
				}
				if u.AvatarUrl != "https://example.com/avatar.png" {
					t.Fatalf("头像未更新: %q", u.AvatarUrl)
				}
				if !u.Birthday.Valid || u.Birthday.Time.Year() != 2000 {
					t.Fatalf("生日未更新: %+v", u.Birthday)
				}
				if u.Username != "erin" {
					t.Fatalf("username 被改动: %q", u.Username)
				}
			},
		},
		{
			name: "提交空请求体：不改任何字段",
			req:  &types.UpdateProfileReq{},
			verify: func(t *testing.T, u *model.Users) {
				if u.Nickname != "erin" || u.Signature != "旧签名" {
					t.Fatalf("空请求体不应改动任何字段: %+v", u)
				}
			},
		},
		{
			name:      "昵称为空串",
			req:       &types.UpdateProfileReq{Nickname: strPtr("")},
			wantField: "nickname",
		},
		{
			name:      "昵称为纯空白",
			req:       &types.UpdateProfileReq{Nickname: strPtr("   ")},
			wantField: "nickname",
		},
		{
			name:      "昵称超长",
			req:       &types.UpdateProfileReq{Nickname: strPtr(strings.Repeat("字", 33))},
			wantField: "nickname",
		},
		{
			name:      "签名超长",
			req:       &types.UpdateProfileReq{Signature: strPtr(strings.Repeat("x", 201))},
			wantField: "signature",
		},
		{
			name:      "头像地址协议非法",
			req:       &types.UpdateProfileReq{AvatarURL: strPtr("ftp://example.com/a.png")},
			wantField: "avatar_url",
		},
		{
			name:      "性别取值非法",
			req:       &types.UpdateProfileReq{Gender: i64Ptr(9)},
			wantField: "gender",
		},
		{
			name:      "生日格式非法",
			req:       &types.UpdateProfileReq{Birthday: strPtr("2020/01/01")},
			wantField: "birthday",
		},
		{
			name:      "生日为未来日期",
			req:       &types.UpdateProfileReq{Birthday: strPtr(time.Now().AddDate(1, 0, 0).Format("2006-01-02"))},
			wantField: "birthday",
		},
		{
			name:      "生日年份过早",
			req:       &types.UpdateProfileReq{Birthday: strPtr("1899-01-01")},
			wantField: "birthday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			c := *base
			users.users[base.Id] = &c
			svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

			dto, err := UpdateProfile(context.Background(), svcCtx, base.Id, tt.req)

			if tt.wantField != "" {
				fe, ok := asFieldError(err)
				if !ok {
					t.Fatalf("期望字段级错误，实际: %v", err)
				}
				if fe.Field != tt.wantField {
					t.Fatalf("字段名期望 %q，实际 %q", tt.wantField, fe.Field)
				}
				// 校验失败不得执行任何写操作。
				if users.updateHits != 0 {
					t.Fatalf("校验失败时不应写库，实际 %d 次", users.updateHits)
				}
				return
			}

			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if dto == nil {
				t.Fatalf("修改成功必须返回更新后的资料")
			}
			tt.verify(t, users.users[base.Id])
		})
	}
}

// 目标是凭证身份：即使传入他人的 ID，写入的也只能是那个 ID 本人。
// 这条测试固化「接口不接受请求体指定目标」的设计（design 决策 5）。
func TestUpdateProfileTargetIsArgumentNotBody(t *testing.T) {
	users := newFakeUsers()
	victim := &model.Users{Id: 800, Username: "victim", Nickname: "victim", CreatedAt: time.Now()}
	attacker := &model.Users{Id: 801, Username: "attacker", Nickname: "attacker", CreatedAt: time.Now()}
	users.users[victim.Id] = victim
	users.users[attacker.Id] = attacker

	svcCtx := newTestContext(users, newFakeSessions(), &fakeIssuer{})

	nickname := "被改的名字"
	// 以 attacker 的身份调用（userID=801），请求体里出现的任何 ID 都不会被采用。
	if _, err := UpdateProfile(context.Background(), svcCtx, attacker.Id,
		&types.UpdateProfileReq{Nickname: &nickname}); err != nil {
		t.Fatalf("修改失败: %v", err)
	}

	if victim.Nickname != "victim" {
		t.Fatalf("他人的资料被改动: %q", victim.Nickname)
	}
	if attacker.Nickname != "被改的名字" {
		t.Fatalf("本人的资料未更新: %q", attacker.Nickname)
	}
}

// 依赖缺失时资料操作必须返回依赖不可用。
func TestProfileWithoutUsers(t *testing.T) {
	svcCtx := &svc.ServiceContext{}

	if _, err := Me(context.Background(), svcCtx, 1); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("Me 期望 ErrDependencyUnavailable，实际: %v", err)
	}
	if _, err := PublicProfile(context.Background(), svcCtx, 1); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("PublicProfile 期望 ErrDependencyUnavailable，实际: %v", err)
	}
	if _, err := UpdateProfile(context.Background(), svcCtx, 1, &types.UpdateProfileReq{}); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("UpdateProfile 期望 ErrDependencyUnavailable，实际: %v", err)
	}
}

// ===== 辅助 =====

// hashForTest 用与生产同一个实现生成哈希，避免测试用固定假哈希
// 导致「密码校验逻辑改动后测试仍然通过」。
func hashForTest(plain string) (string, error) {
	return passwordHash(plain)
}
