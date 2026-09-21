package handler

import (
	"net/http"

	"bilibili-web/server/app/user/api/internal/binder"
	"bilibili-web/server/app/user/api/internal/logic"
	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/app/user/api/internal/types"
	"bilibili-web/server/common/response"
)

// routeHandlers 声明本域全部路由的记录位置，便于评审时一眼看全。
//
// 六条路由的登录要求由是否使用 protected 包装决定（design 决策 5）：
//
//	POST /api/v1/user/register      公开
//	POST /api/v1/user/login         公开
//	POST /api/v1/user/logout        受保护
//	GET  /api/v1/user/me            受保护
//	PUT  /api/v1/user/me            受保护
//	GET  /api/v1/user/users/{id}    公开
func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteOK(w, logic.Health(svcCtx))
	}
}

// RegisterHandler 处理 POST /api/v1/user/register。
func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := binder.Decode(r, &req); err != nil {
			writeBindError(w, err)
			return
		}

		result, err := logic.Register(r.Context(), svcCtx, &req)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		// 响应只含 token 与 DTO；DTO 由白名单转换函数构造，不含 password_hash。
		response.WriteOK(w, types.RegisterData{Token: result.Token, User: result.User})
	}
}

// LoginHandler 处理 POST /api/v1/user/login。
func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := binder.Decode(r, &req); err != nil {
			writeBindError(w, err)
			return
		}

		result, err := logic.Login(r.Context(), svcCtx, &req)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		response.WriteOK(w, types.LoginData{Token: result.Token, User: result.User})
	}
}

// LogoutHandler 处理 POST /api/v1/user/logout。
//
// 本接口受保护：未登录时无从确定要撤销谁的会话，
// 且「登出」在无凭证时本就无事可做，返回 40100 是准确的语义。
func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := identityOf(r)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		if err := logic.Logout(r.Context(), svcCtx, identity.UserID); err != nil {
			writeLogicError(w, err)
			return
		}

		response.WriteOK(w, nil)
	}
}

// MeHandler 处理 GET /api/v1/user/me。
func MeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := identityOf(r)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		user, err := logic.Me(r.Context(), svcCtx, identity.UserID)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		response.WriteOK(w, user)
	}
}

// UpdateMeHandler 处理 PUT /api/v1/user/me。
func UpdateMeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := identityOf(r)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		// 先取顶层字段名做白名单检查，再解码到结构体。
		//
		// 顺序很重要：identity/profile 要求「请求体中出现白名单外字段时
		// MUST 整体拒绝，MUST NOT 静默忽略后继续执行部分更新」。
		// 先整体检查可以一次性告知全部越界字段，而不是解码报错时只报第一个。
		fields, err := binder.RawFields(r)
		if err != nil {
			writeBindError(w, err)
			return
		}
		if err := binder.RejectUnknownFields(fields, profileWritableFields...); err != nil {
			writeBindError(w, err)
			return
		}

		req := &types.UpdateProfileReq{}
		if err := decodeFieldsInto(fields, req); err != nil {
			writeBindError(w, err)
			return
		}

		user, err := logic.UpdateProfile(r.Context(), svcCtx, identity.UserID, req)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		response.WriteOK(w, user)
	}
}

// PublicProfileHandler 处理 GET /api/v1/user/users/{id}。
//
// 公开接口：无需登录，只返回公开字段。
func PublicProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := pathInt64(r, "id")
		if err != nil {
			writeBindError(w, err)
			return
		}

		user, err := logic.PublicProfile(r.Context(), svcCtx, userID)
		if err != nil {
			writeLogicError(w, err)
			return
		}

		response.WriteOK(w, user)
	}
}
