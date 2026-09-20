# Proposal

## Why

骨架变更已把五个业务域建齐，但每个域只有一个 health 接口，没有任何业务能力。鉴权是所有用户相关功能（个人空间、投稿、点赞、评论）的公共前置，也是 `identity/auth` 基线规范里「受保护接口服务端强制校验」唯一尚未落地的要求——当前 `common/middleware/Auth` 是个直通空壳，受保护接口在服务端实际上没有任何校验。

更关键的是一个被低估的事实：**这是项目里第一个真正落库的功能，而数据库地基完全不存在**。`server/go.mod` 里没有 MySQL 驱动、没有 Redis 客户端，`app/user/model/` 只有一个 README，各域配置文件里没有任何存储连接项。如果不把这套地基作为独立阶段先立起来，它会在实现注册、登录、改资料时被反复以临时方式触碰，最终散落在各处。现在做的成本最低：骨架刚落地、业务代码为零。

## What Changes

**第一阶段：后端基础设施首建（独立成阶段，先于任何鉴权业务代码）**

- 在 `server/` 引入 MySQL 驱动与 Redis 客户端依赖，并在 `user` 域建立可复用的连接建立与生命周期管理。
- 扩展域服务配置结构，新增 MySQL、Redis、JWT 密钥三类配置项，全部通过 `etc/*.yaml` + 环境变量注入，禁止硬编码。
- 落地 `app/user/model/` 数据访问层（当前为空目录），建立 `users` 表与建表脚本，脚本置于 `deploy/`。
- 交付可验证的地基验收：服务启动即连通 MySQL 与 Redis，health 接口的 `mysql` / `redis` 依赖项由「探测端口」升级为「真实连接」。

**第二阶段：用户鉴权能力**

- 新增注册、登录、退出登录三个能力，凭证采用 JWT（HS256），服务端签发。
- 新增个人信息展示与修改：查询当前登录用户资料、修改本人资料、查询他人公开资料。
- 密码使用 bcrypt 哈希存储，任何响应均不得包含密码哈希。
- 退出登录 SHALL 使该凭证在服务端即刻失效，而非仅由前端清除本地存储。
- 将 `common/middleware/Auth` 由直通空壳替换为真实校验，供全部五个域复用。

**第三阶段：前端接入**

- 登录页、注册页、个人资料页三个页面。
- `api/http.ts` 统一注入凭证并在收到未认证错误码时清除登录态、引导重新登录。
- 路由层新增受保护路由守卫，未登录访问受限页面时跳转登录并保留原目标地址。
- 新增登录态 Zustand 切片，承载跨页面共享的凭证与当前用户信息。

**明确不做（Out of Scope）**

- 单点登录（跨应用 SSO）、扫码登录、第三方 OAuth 登录。
- 短信/邮箱验证码、邮箱验证激活。
- refresh token 与自动续期；仅签发单一凭证并设置固定有效期。
- 头像文件上传与存储（头像字段仅接受外部 URL 字符串）。
- 用户等级、权限角色的业务实现（`identity/auth` 已确立模型，本变更仅落字段与默认值）。

## Capabilities

### New Capabilities

- `identity/profile`: 个人信息的展示与修改能力，含字段约束、本人与他人的可见性差异、联系方式脱敏规则，以及修改操作的归属校验。

### Modified Capabilities

- `identity/auth`: 新增注册、登录态撤销（退出登录）与凭证携带方式的具体要求。既有五条 requirement（登录态签发与携带、凭证过期统一处理、受保护接口服务端强制校验、用户等级与权限模型、敏感信息不得外泄）保持不变，本变更以 ADDED delta 增量补充，不修改既有内容。

## 触及的基线规范

本变更触及以下 `openspec/specs/` 下的能力，逐条核对结论见 design.md：

| 基线能力 | 触及方式 | 是否违反 |
| --- | --- | --- |
| `identity/auth` | 以 ADDED delta 新增注册、撤销与凭证要求 | 否，仅增量补充 |
| `identity/profile` | 新增能力，无既有约束 | 否 |
| `api/contract` | 新接口须遵守统一响应、错误码区段、参数校验位置 | 否 |
| `backend/architecture` | 新代码须落在 api → rpc → model 分层；认证中间件须落在 `common/` | 否 |
| `data/platform` | users 表建模、Snowflake 对外 ID、软删除、Redis 键规范 | 否 |
| `frontend/conventions` | 新增页面、store 切片、类型与接口封装的落位 | 否 |
| `quality/testing` | 每个任务须携带可验证验收步骤 | 否 |

## Impact

- **新增依赖**：MySQL 驱动、Redis 客户端、bcrypt 所在库、JWT 库。其中 `github.com/golang-jwt/jwt/v4` 已作为 go-zero 间接依赖存在于 `go.sum`，但本变更选用当前主版本 v5，需显式 `go get` 并 `go mod tidy`。
- **新增数据库对象**：`bilibili_web` 库下的 `users` 表及建表脚本，脚本置于 `server/deploy/`。
- **配置文件变更**：五个域的 `etc/*.yaml` 与 `internal/config/config.go` 均需扩展存储与密钥配置项。
- **公共层变更**：`server/common/middleware/middleware.go` 的 `Auth` 由空壳替换为真实校验，影响全部五个域的路由注册方式。
- **前端变更**：`web/src/api/http.ts`、`web/src/app/router.tsx`、`web/src/store/`、`web/src/types/`、`web/src/pages/`。
- **接口新增**：`/api/v1/user/` 下新增 6 个接口（注册、登录、退出、查本人资料、改本人资料、查他人公开资料）。
- **不涉及**：视频、互动、评论、搜索四域的业务逻辑；但四域的受保护接口将从本变更起获得真实鉴权能力。
