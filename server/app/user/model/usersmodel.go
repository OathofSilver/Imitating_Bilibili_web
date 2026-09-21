package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UsersModel = (*customUsersModel)(nil)

// ProfileUpdate 是个人资料的可修改字段白名单。
//
// identity/profile 明确规定：用户名、用户 ID、等级、角色不得通过资料修改接口变更。
// 因此这里单独暴露可改字段，而不是复用生成的 Update——后者会更新除 id 与时间字段外的
// 全部列（包含 username），用它改资料会写坏用户名。
type ProfileUpdate struct {
	Nickname  string
	AvatarUrl string
	Signature string
	Gender    int64
	Birthday  sql.NullTime
}

type (
	// UsersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUsersModel.
	UsersModel interface {
		usersModel
		withSession(session sqlx.Session) UsersModel

		// UpdateProfile 按白名单更新个人资料，不触碰 username 等不可变字段。
		UpdateProfile(ctx context.Context, id int64, in ProfileUpdate) error
	}

	customUsersModel struct {
		*defaultUsersModel
	}
)

// NewUsersModel returns a model for the database table.
func NewUsersModel(conn sqlx.SqlConn) UsersModel {
	return &customUsersModel{
		defaultUsersModel: newUsersModel(conn),
	}
}

func (m *customUsersModel) withSession(session sqlx.Session) UsersModel {
	return NewUsersModel(sqlx.NewSqlConnFromSession(session))
}

// FindOne 覆盖生成实现，追加软删除过滤。
//
// goctl 生成的语句只按主键查询，会返回已软删除的用户，使
// identity/profile「目标用户已被删除时返回 40400」无法成立。
func (m *customUsersModel) FindOne(ctx context.Context, id int64) (*Users, error) {
	query := fmt.Sprintf("select %s from %s where `id` = ? and `deleted_at` is null limit 1", usersRows, m.table)

	var resp Users
	err := m.conn.QueryRowCtx(ctx, &resp, query, id)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("model: 按 id 查询用户失败: %w", err)
	}
}

// FindOneByUsername 覆盖生成实现，追加软删除过滤。
func (m *customUsersModel) FindOneByUsername(ctx context.Context, username string) (*Users, error) {
	query := fmt.Sprintf("select %s from %s where `username` = ? and `deleted_at` is null limit 1", usersRows, m.table)

	var resp Users
	err := m.conn.QueryRowCtx(ctx, &resp, query, username)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("model: 按用户名查询用户失败: %w", err)
	}
}

// Delete 覆盖生成实现，改为软删除。
//
// goctl 生成的是物理 delete，违反 data/platform「业务实体的删除 SHALL 采用软删除，
// MUST NOT 物理删除业务数据」。这里选择覆盖而非弃用，是为了消除「被误调用的物理删除入口」。
func (m *customUsersModel) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("update %s set `deleted_at` = now() where `id` = ? and `deleted_at` is null", m.table)

	if _, err := m.conn.ExecCtx(ctx, query, id); err != nil {
		return fmt.Errorf("model: 软删除用户失败: %w", err)
	}

	return nil
}

// UpdateProfile 按白名单更新个人资料，仅允许改动昵称、头像地址、签名、性别与生日。
func (m *customUsersModel) UpdateProfile(ctx context.Context, id int64, in ProfileUpdate) error {
	query := fmt.Sprintf(
		"update %s set `nickname` = ?, `avatar_url` = ?, `signature` = ?, `gender` = ?, `birthday` = ? "+
			"where `id` = ? and `deleted_at` is null",
		m.table,
	)

	if _, err := m.conn.ExecCtx(ctx, query, in.Nickname, in.AvatarUrl, in.Signature, in.Gender, in.Birthday, id); err != nil {
		return fmt.Errorf("model: 更新用户资料失败: %w", err)
	}

	return nil
}
