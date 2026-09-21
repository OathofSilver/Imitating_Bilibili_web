package logic

import (
	"context"
	"fmt"

	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/app/user/api/internal/types"
	"bilibili-web/server/app/user/model"
)

// ErrUserNotFound 表示目标用户不存在或已被软删除。
var ErrUserNotFound = fmt.Errorf("用户不存在")

// Me 返回指定用户的本人资料。
//
// userID 由鉴权中间件从凭证中解出，不接受请求参数指定——
// 这是「修改与查询的目标恒为凭证身份」的实现基础（identity/profile）。
func Me(ctx context.Context, svcCtx *svc.ServiceContext, userID int64) (*types.UserDTO, error) {
	if svcCtx.Users == nil {
		return nil, ErrDependencyUnavailable
	}

	user, err := svcCtx.Users.FindOne(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			// 凭证有效但用户已被软删除：这在正常流程下不该出现，
			// 因此按「资源不存在」处理并记录日志以便发现异常。
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("查询本人资料: %w", err)
	}

	return userToDTO(user), nil
}

// PublicProfile 返回指定用户的公开资料，无需登录。
func PublicProfile(ctx context.Context, svcCtx *svc.ServiceContext, userID int64) (*types.PublicUserDTO, error) {
	if svcCtx.Users == nil {
		return nil, ErrDependencyUnavailable
	}
	if userID <= 0 {
		return nil, ErrUserNotFound
	}

	// FindOne 已在 model 层带 `deleted_at IS NULL` 过滤，
	// 因此已软删除的用户在这里会得到 ErrNotFound 而非残缺对象
	// （identity/profile 的「目标用户不存在或已被删除」场景）。
	user, err := svcCtx.Users.FindOne(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("查询公开资料: %w", err)
	}

	return userToPublicDTO(user), nil
}

// UpdateProfile 修改本人资料。
//
// 修改范围限定为昵称、头像地址、签名、性别、生日五个白名单字段，
// 目标恒为 userID（来自凭证），请求体中出现的其他字段在 binder 层
// 已被整体拒绝，model 层用的是白名单 UPDATE 而非生成的 Update。
//
// 做法是「先读后写」而非「直接 UPDATE 提交的字段」：
// 接口语义是「返回更新后的完整资料」，因此无论改了哪些字段都要回读一次；
// 同时回读也顺带验证了目标用户确实存在且未被删除。
func UpdateProfile(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	userID int64,
	req *types.UpdateProfileReq,
) (*types.UserDTO, error) {
	if svcCtx.Users == nil {
		return nil, ErrDependencyUnavailable
	}

	current, err := svcCtx.Users.FindOne(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("修改资料: 读取当前资料失败: %w", err)
	}

	// 以当前值为基底，只覆盖请求中显式提交的字段。
	// 用「完整对象 + 整体 UPDATE」而非「按字段拼 SQL」，
	// 使白名单在类型层面就是封闭的——无法通过构造请求体写入其他列。
	next := model.ProfileUpdate{
		Nickname:  current.Nickname,
		AvatarUrl: current.AvatarUrl,
		Signature: current.Signature,
		Gender:    current.Gender,
		Birthday:  current.Birthday,
	}

	if req.Nickname != nil {
		value := *req.Nickname
		if err := validateNickname(value); err != nil {
			return nil, &FieldError{Field: "nickname", Err: err}
		}
		next.Nickname = trimSpace(value)
	}

	if req.AvatarURL != nil {
		value := trimSpace(*req.AvatarURL)
		if err := validateAvatarURL(value); err != nil {
			return nil, &FieldError{Field: "avatar_url", Err: err}
		}
		next.AvatarUrl = value
	}

	if req.Signature != nil {
		value := *req.Signature
		if err := validateSignature(value); err != nil {
			return nil, &FieldError{Field: "signature", Err: err}
		}
		next.Signature = trimSpace(value)
	}

	if req.Gender != nil {
		if err := validateGender(*req.Gender); err != nil {
			return nil, &FieldError{Field: "gender", Err: err}
		}
		next.Gender = *req.Gender
	}

	if req.Birthday != nil {
		birthday, err := parseBirthday(*req.Birthday)
		if err != nil {
			return nil, &FieldError{Field: "birthday", Err: err}
		}
		next.Birthday = birthday
	}

	if err := svcCtx.Users.UpdateProfile(ctx, userID, next); err != nil {
		return nil, fmt.Errorf("修改资料: 写入失败: %w", err)
	}

	updated, err := svcCtx.Users.FindOne(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("修改资料: 回读失败: %w", err)
	}

	return userToDTO(updated), nil
}
