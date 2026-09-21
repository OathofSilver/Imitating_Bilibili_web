package types

// HealthData 与本域健康检查响应体一致，字段与统一响应中的 data 对应。
type HealthData struct {
	Service      string          `json:"service"`
	Dependencies map[string]bool `json:"dependencies"`
}

// ===== 注册 =====

// RegisterReq 是注册请求体。
type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterData 是注册响应体。
//
// 直接返回凭证，使注册成功即进入登录态，无需二次登录（identity/auth）。
type RegisterData struct {
	Token string   `json:"token"`
	User  *UserDTO `json:"user"`
}

// ===== 登录 =====

// LoginReq 是登录请求体。
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginData 是登录响应体。
type LoginData struct {
	Token string   `json:"token"`
	User  *UserDTO `json:"user"`
}

// ===== 资料 =====

// UserDTO 是本人资料的响应结构。
//
// 字段与 users 表一一对应，但**刻意不含 PasswordHash**：
// 用独立的 DTO 而非直接序列化 model.Users 是这里最重要的设计——
// 后者带 `db` 标签而无 `json` 标签，一旦有人给它补上 json 标签，
// 密码哈希就会随响应外泄。DTO 让「不返回什么」成为显式代码。
type UserDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Signature string `json:"signature"`
	Gender    int64  `json:"gender"`
	Birthday  string `json:"birthday"`
	Level     int64  `json:"level"`
	Role      int64  `json:"role"`
	CreatedAt string `json:"created_at"`
}

// PublicUserDTO 是他人公开资料的响应结构。
//
// 只含公开字段：不含 username（登录名属于账号信息）、birthday、
// created_at 等内部字段，也不含任何联系方式（identity/profile）。
type PublicUserDTO struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Signature string `json:"signature"`
	Gender    int64  `json:"gender"`
	Level     int64  `json:"level"`
}

// UpdateProfileReq 是资料修改请求体。
//
// 全部字段用指针，以区分「未提交」（nil）与「提交为空值」（指向零值）。
//
// 白名单外的字段刻意**不出现在本结构上**：若请求体带了 username、id、level
// 或 role，会因为结构体没有对应字段而在解码阶段被识别为未知字段，
// 由 logic 层显式拒绝（identity/profile 要求整体拒绝，不得静默忽略）。
type UpdateProfileReq struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Signature *string `json:"signature"`
	Gender    *int64  `json:"gender"`
	Birthday  *string `json:"birthday"`
}
