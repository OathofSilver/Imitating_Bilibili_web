package logic

import (
	"context"
	"errors"
	"fmt"

	"bilibili-web/server/app/user/api/internal/password"
	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/app/user/api/internal/types"
	"bilibili-web/server/app/user/model"
	"bilibili-web/server/common/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

// ErrInvalidCredentials 是「用户名不存在」与「密码错误」的统一错误。
//
// 两种情况刻意共用一个错误：identity/auth 明确规定 MUST NOT 通过不同的
// 错误码或文案区分二者，否则接口就成了账号枚举工具。
var ErrInvalidCredentials = errors.New("用户名或密码错误")

// ErrUsernameTaken 表示用户名已被占用。
var ErrUsernameTaken = errors.New("该用户名已被占用")

// ErrDependencyUnavailable 表示本接口依赖的存储或鉴权组件不可用。
//
// 与具体的 model 错误分开，是为了让 handler 能干净地映射为 50001：
// 「依赖挂了」和「业务规则不通过」对客户端的含义完全不同，
// 前者值得重试，后者重试永远不会成功。
var ErrDependencyUnavailable = errors.New("服务依赖不可用")

// AuthResult 是注册与登录的共同产出。
type AuthResult struct {
	Token string
	User  *types.UserDTO
}

// Register 完成用户注册：
// 校验参数 → bcrypt 哈希 → Snowflake 生成 ID → 写库 → 递增会话版本 → 签发凭证。
//
// 注册成功直接返回凭证，用户无需二次登录（identity/auth）。
func Register(ctx context.Context, svcCtx *svc.ServiceContext, req *types.RegisterReq) (*AuthResult, error) {
	if svcCtx.Users == nil || svcCtx.Tokens == nil || svcCtx.Sessions == nil {
		return nil, ErrDependencyUnavailable
	}

	username := normalizeUsername(req.Username)
	if err := validateUsername(username); err != nil {
		return nil, &FieldError{Field: "username", Err: err}
	}
	// 密码强度的校验放在 password.Hash 内部统一做，避免两处规则不一致；
	// 这里先调用 Validate 以便把错误归到 password 字段上。
	if err := password.Validate(req.Password); err != nil {
		return nil, &FieldError{Field: "password", Err: err}
	}

	hashed, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("注册: 生成密码哈希失败: %w", err)
	}

	id, err := snowflake.NextID()
	if err != nil {
		return nil, fmt.Errorf("注册: 生成用户 ID 失败: %w", err)
	}

	// 新用户的资料具有确定默认值：昵称取用户名（identity/profile 要求
	// MUST NOT 出现空昵称），其余未填写字段为约定空值。
	user := &model.Users{
		Id:           id,
		Username:     username,
		PasswordHash: hashed,
		Nickname:     username,
		AvatarUrl:    "",
		Signature:    "",
		Gender:       genderUnknown,
		Level:        defaultLevel,
		Role:         defaultRole,
	}

	if _, err := svcCtx.Users.Insert(ctx, user); err != nil {
		// 唯一索引是防重的最终防线：并发下「先查后插」存在竞态，
		// 这里的错误码判定必须保留（identity/auth 的注册场景要求）。
		if model.IsDuplicateUsername(err) {
			return nil, ErrUsernameTaken
		}

		return nil, fmt.Errorf("注册: 写入用户失败: %w", err)
	}

	// 回读一次，让数据库填充的字段真实落在响应里。
	//
	// `created_at`/`updated_at` 由表默认值生成，goctl 生成的 Insert 又刻意
	// 不带这两列，因此上面那个内存结构体的 CreatedAt 仍是 Go 零值，
	// 直接转换会让响应返回 `0001-01-01T00:00:00Z`。
	// 回读还有一个附带好处：注册响应与 /me 的字段来源完全一致。
	created, err := svcCtx.Users.FindOne(ctx, user.Id)
	if err != nil {
		return nil, fmt.Errorf("注册: 回读新用户失败: %w", err)
	}

	// 递增会话版本号并把它写进凭证：这样同一用户先前签发的凭证会立即失效，
	// 形成「新登录踢掉旧登录」的效果，也让退出登录有据可依。
	ver, err := svcCtx.Sessions.Bump(ctx, user.Id)
	if err != nil {
		return nil, fmt.Errorf("注册: 初始化会话版本失败: %w", err)
	}

	raw, err := svcCtx.Tokens.Issue(user.Id, ver, nowFunc())
	if err != nil {
		return nil, fmt.Errorf("注册: 签发凭证失败: %w", err)
	}

	return &AuthResult{Token: raw, User: userToDTO(created)}, nil
}

// Login 校验用户名与密码，成功时递增会话版本并签发新凭证。
func Login(ctx context.Context, svcCtx *svc.ServiceContext, req *types.LoginReq) (*AuthResult, error) {
	if svcCtx.Users == nil || svcCtx.Tokens == nil || svcCtx.Sessions == nil {
		return nil, ErrDependencyUnavailable
	}

	// 用户名归一化后交给唯一索引查询；不预先校验格式，
	// 否则「用户名格式非法」与「用户名不存在」的响应就会不同，形成旁路。
	username := normalizeUsername(req.Username)

	user, err := svcCtx.Users.FindOneByUsername(ctx, username)
	if err != nil {
		if isNotFound(err) {
			// 刻意与密码错误返回同一个错误，且不记 warning 级日志以外的信息，
			// 避免日志侧把「不存在的用户名」变成可枚举的侧信道。
			logx.Infof("[user] 登录失败：用户名不存在")
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("登录: 查询用户失败: %w", err)
	}

	if !password.Verify(user.PasswordHash, req.Password) {
		logx.Infof("[user] 登录失败：密码不匹配 uid=%d", user.Id)
		return nil, ErrInvalidCredentials
	}

	ver, err := svcCtx.Sessions.Bump(ctx, user.Id)
	if err != nil {
		return nil, fmt.Errorf("登录: 递增会话版本失败: %w", err)
	}

	raw, err := svcCtx.Tokens.Issue(user.Id, ver, nowFunc())
	if err != nil {
		return nil, fmt.Errorf("登录: 签发凭证失败: %w", err)
	}

	return &AuthResult{Token: raw, User: userToDTO(user)}, nil
}

// Logout 递增会话版本号，使该用户已签发的全部凭证立即失效。
//
// 只递增而不记录具体凭证：版本号机制下，撤销的粒度是「该用户的所有凭证」，
// 这与「简单鉴权」的目标一致——不做多端登录管理（design 决策 2）。
func Logout(ctx context.Context, svcCtx *svc.ServiceContext, userID int64) error {
	if svcCtx.Sessions == nil {
		return ErrDependencyUnavailable
	}
	if userID <= 0 {
		return ErrDependencyUnavailable
	}

	if _, err := svcCtx.Sessions.Bump(ctx, userID); err != nil {
		// 会话存储故障归入依赖不可用而非内部错误：前者值得客户端重试，
		// 且 handler 会把它映射为 50001 并记录原因（design Risks）。
		logx.Errorf("[user] 退出登录失败：递增会话版本出错 uid=%d: %v", userID, err)
		return fmt.Errorf("%w: 递增会话版本失败", ErrDependencyUnavailable)
	}

	return nil
}
