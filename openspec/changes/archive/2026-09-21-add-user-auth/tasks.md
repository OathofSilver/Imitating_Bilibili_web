# Tasks

## 1. 后端基础设施首建（第一阶段，先于一切鉴权业务代码）

本阶段不产出任何对外接口变化，可独立验证、独立提交。

- [x] 1.1 引入四项依赖并整理依赖树：`github.com/golang-jwt/jwt/v5` v5.3.1、`github.com/go-sql-driver/mysql` v1.10.1、`github.com/redis/go-redis/v9` v9.22.0、`golang.org/x/crypto` v0.57.0，执行 `go mod tidy`（验收：`go build ./...` 成功，且四项在 `go.mod` 中为直接依赖而非 `// indirect`）
- [x] 1.2 在 `server/common/store/` 新增 MySQL 与 Redis 的连接构造与关闭能力，MySQL 使用 `sqlx.NewConn(sqlx.SqlConf{...})` 构造（不使用 `MustNewConn` / `NewMysql`，避免失败即终止进程）（验收：`go vet ./common/...` 通过；连接失败时返回带上下文的错误而非 panic，且失败分支有单元测试覆盖）
- [x] 1.3 扩展五个域的 `internal/config/config.go` 与 `etc/*.yaml`，新增 MySQL、Redis、JWT 密钥三类配置项，允许被 `MYSQL_ADDR`、`REDIS_ADDR` 等环境变量覆盖（验收：五个域服务均能启动；修改配置后行为随之改变；在 `server/` 内检索不到明文连接串或密钥字面量）
- [x] 1.4 在 `server/deploy/` 新增可重复执行的建表脚本，创建 `users` 表：`id`（Snowflake，主键，非自增）、`username`（唯一索引）、`password_hash`、`nickname`、`avatar_url`、`signature`、`gender`、`birthday`、`level`、`role`、`created_at`、`updated_at`、`deleted_at`，字符集 `utf8mb4`（验收：脚本连续执行两次均成功；`SHOW CREATE TABLE users` 含三类时间字段与 `username` 唯一索引，且语句中无 `AUTO_INCREMENT`；该脚本能被 goctl 的 DDL 解析器接受，即直接用于任务 1.5 的生成命令）
- [x] 1.5 用 goctl 生成 model 层骨架：`goctl model mysql ddl -src deploy/users.sql -dir app/user/model`，**不传 `-c`**（验收：生成 `usersmodel.go`、`usersmodel_gen.go`、`vars.go` 三个文件且包名为 `model`，`go build ./...` 通过；结构体中 `deleted_at` 映射为 `sql.NullTime`；已由唯一索引自动生成 `FindOneByUsername`；生成过程输出的两条 warning 未被据此修改表结构，即未加 `AUTO_INCREMENT`、未给 `username` 与 `password_hash` 加 `DEFAULT`）
- [x] 1.6 在 `usersmodel.go` 的 `customUsersModel` 上覆盖生成代码以符合基线规范：`Delete` 改为写 `deleted_at` 的软删除；`FindOne` 与 `FindOneByUsername` 增加 `deleted_at IS NULL` 过滤；新增白名单方法 `UpdateProfile` 供资料修改使用，不使用生成的 `Update`（验收：`go build ./...` 通过；测试覆盖三类场景——按 ID 查询已软删除用户返回 `ErrNotFound`、软删除后该行仍在库中且 `deleted_at` 非空、用 `UpdateProfile` 改昵称后 `username` 保持原值；重新执行任务 1.5 的生成命令后，自定义方法仍然存在）
- [x] 1.7 将 `common/health` 的依赖状态由 TCP 端口探测升级为真实连接校验：redis 项在五个域均为真实连接，mysql 项在 user 域为真实连接（验收：`make up` 后 `make health` 五域的 redis 均为 true 且 user 域 mysql 为 true；停止 Redis 容器后五域 redis 项变为 false，服务不崩溃且响应仍为统一结构）

## 2. 用户鉴权后端能力（第二阶段）

- [x] 2.1 在 `common/errcode` 新增 `40901`（用户名已存在）并登记文案，保持 `4xxxx` 段归属（验收：单元测试断言 `Message(40901)` 返回非空且不同于 `40001` 的文案）
- [x] 2.2 在 `common/middleware` 实现真实 `Auth`：解析 `Authorization: Bearer`、本地验签与有效期校验、通过撤销检查器校验会话版本号，失败时返回对应错误码（验收：单元测试覆盖缺失请求头、格式错误、签名被篡改、已过期、版本不匹配五类失败与一类成功）
- [x] 2.3 实现凭证的签发与解析能力（HS256，载荷含 `uid`、`ver`、`iat`、`exp`），密钥取自配置（验收：单元测试覆盖签发后可解析、篡改签名被拒、超过有效期被拒）
- [x] 2.4 实现会话版本号的 Redis 存储操作：读取当前版本、递增版本，键为 `bw:v1:auth:ver:<uid>` 并设置过期时间（验收：调用后 `redis-cli TTL bw:v1:auth:ver:<uid>` 返回大于 0 的值；键名符合 `data/platform` 的前缀 + 结构版本规范）
- [x] 2.5 实现注册接口 `POST /api/v1/user/register`：参数校验、bcrypt 哈希、Snowflake 生成 ID、写入默认资料、直接签发凭证；用户名冲突由 `username` 唯一索引兜底，model 层判定 MySQL 错误码 1062 后由 api 层映射为 `40901`（验收：`curl` 注册返回 `code` 为 0 且 data 含凭证；同名重复注册返回 `40901`；响应中检索不到 `password_hash`）
- [x] 2.6 实现登录接口 `POST /api/v1/user/login`：校验用户名与密码，成功时递增会话版本号并签发凭证（验收：正确凭据返回 `code` 为 0；错误密码与不存在的用户名返回完全相同的 `40100` 与相同文案）
- [x] 2.7 实现退出登录接口 `POST /api/v1/user/logout`：递增该用户会话版本号（验收：登出后携带原凭证请求 `/api/v1/user/me` 返回 `40100`）
- [x] 2.8 实现 `GET /api/v1/user/me` 与 `PUT /api/v1/user/me`：本人资料查询与修改，修改范围限昵称、头像地址、签名、性别、生日白名单，目标恒为凭证身份（验收：修改昵称后再次查询返回新值；请求体含 `username`、`id`、`level` 或 `role` 时返回 `40001` 且无任何字段被更新）
- [x] 2.9 实现 `GET /api/v1/user/users/{id}`：他人公开资料，无需登录，仅返回昵称、头像地址、签名、性别、等级（验收：不带凭证访问返回 `code` 为 0 且响应字段仅含公开项；对已软删除用户返回 `40400`）
- [x] 2.10 在 user 域 `api` 层注册六个路由，受保护接口挂载 `middleware.Auth`（验收：逐个 `curl`，三个公开接口未登录可通，三个受保护接口未登录返回 `40100`）
- [x] 2.11 为注册、登录、资料修改的 logic 层编写 table-driven 测试，外部依赖以接口注入（验收：`go test ./app/user/...` 通过，且覆盖成功与失败两条路径）

## 3. 前端接入（第三阶段）

- [x] 3.1 在 `web/src/types/` 定义用户与鉴权相关的请求与响应类型（验收：`pnpm tsc --noEmit` 通过且无 `any`）
- [x] 3.2 改造 `web/src/api/http.ts`：统一注入 `Authorization` 头，并在收到 `40100` 时清除本地凭证并引导重新登录（验收：请求头可观察到携带凭证；将本地凭证替换为过期值后调用受保护接口，页面跳转登录而非停留在空白状态）
- [x] 3.3 在 `web/src/api/` 封装六个用户接口（验收：`pages/` 与 `features/` 内检索不到直接调用的 `fetch`）
- [x] 3.4 新增登录态 Zustand 切片，承载凭证与当前用户（验收：登录后刷新页面登录态仍可恢复；`pnpm tsc --noEmit` 通过）
- [x] 3.5 新增注册页与登录页，表单组件置于 `features/`（验收：注册成功后直接进入登录态；用户名被占用时展示明确提示而非通用错误；样式取自 Mantine 主题 token，无硬编码色值）
- [x] 3.6 新增个人资料页：展示本人资料并支持编辑提交（验收：修改昵称提交后页面展示新值，刷新后仍为新值）
- [x] 3.7 在 `web/src/app/router.tsx` 增加受保护路由守卫，未登录访问受限页面时跳转登录并保留原目标地址（验收：未登录访问资料页跳转登录页，登录成功后回到资料页）
- [x] 3.8 在 `AppLayout` 展示登录态：未登录显示登录入口，已登录显示昵称与登出入口（验收：登录后头部显示昵称，点击登出后回到未登录状态）

## 4. 规范核对与整体验收

- [x] 4.1 逐条核对 design.md 中的「触及的基线规范逐条核对」表，确认实现未偏离（验收：核对结论记录在本变更中，列出全部违反项或明确「无违反」）
- [x] 4.2 端到端链路验证：注册 → 登录 → 查询本人资料 → 修改资料 → 登出 → 用旧凭证访问受保护接口被拒（验收：六个步骤结果全部符合预期，且第 6 步返回 `40100`）
- [x] 4.3 执行 `openspec validate add-user-auth --strict`（验收：命令退出码为 0）

## 5. 实施记录（决策与实测结论）

### 5.1 决策 9 的落地（任务 1.3 的明文凭据归零）

原方案在 `MysqlConf` 与五份 yaml 里都保留了 `root` 作为默认值，`usersmodel_test.go` 里还有完整连接串字面量。经确认改为彻底归零：

- `MysqlConf.User` 去掉默认值，账号与口令统一由 `MYSQL_USER` / `MYSQL_PASSWORD` 注入。
- 五份 `etc/*.yaml` 删除 `User: root` 行。
- `deploy/docker-compose.yaml` 的口令改为 `${MYSQL_ROOT_PASSWORD:-${MYSQL_PASSWORD:?...}}`，healthcheck 用 `CMD-SHELL` 引用容器内变量；口令落在 gitignore 的 `deploy/.env`，模板为 `deploy/.env.example`。
- `usersmodel_test.go` 的连接串改为完全由 `BW_TEST_MYSQL_DSN` 提供；`store_test.go` 的假连接串改为 `sampleDSN()` 拼装。
- 补偿：新增 `make test` 目标，由它统一拼出 `BW_TEST_MYSQL_DSN` 与 `MYSQL_*`，使 `go test ./...` 在开发机上仍覆盖 model 层。

**副作用（已确认接受）**：直接跑 `go test ./app/user/model/...` 会因缺少该变量而跳过 4 个用例。开发时使用 `make test`。

### 5.2 实测发现：宿主机 8001-8005 落在 Windows 保留端口段

原配置的 8001-8005 在 `netsh interface ipv4 show excludedportrange protocol=tcp` 中落在 `7925-8024` 与 `8025-8124` 两个排除段内，导致 `bind: An attempt was made to access a socket in a way forbidden by its access permissions` 并 panic。

已将五域端口改为 **18001-18005**，`Makefile` 的 `health` 目标同步更新。这与 `docker-compose.yaml` 用 13306 / 5673 的理由相同（本机保留段与既有服务占用）。

### 5.3 实测发现：go-zero rest 的路径参数不能用标准库 API

`http.Request.PathValue` 在 go-zero v1.10.3 的 rest 下始终返回空串——它用的是自己的路由树，路径参数写入 `rest/pathvar` 的私有上下文键。必须用 `pathvar.Vars(r)`。已修正 `handler/request.go`。

### 5.4 实测发现：goctl 生成的 `Insert` 不回填时间列（结论：无需处理）

`Users.CreatedAt` / `UpdatedAt` 由 MySQL 的 `DEFAULT CURRENT_TIMESTAMP` 生成，而 `Insert` 的列清单里不含这两列。实施中曾观察到注册响应的时间为零值，**但重启服务后重新验证为正确的当前时间**——前一次是对着未替换的旧进程发的请求（端口被旧 PID 占用、新进程 `bind` 失败 panic）。

因此不做额外回读、不自定义 `Insert`。详见 design.md 决策 11（含该误判的教训：验证接口行为前必须确认进程已替换）。

### 5.5 任务 1.1 的依赖版本偏差

`golang.org/x/crypto` 在 `go.mod` 中为 **v0.57.0**（与任务描述一致），`go` 指令为 1.26.0。`go mod tidy` 单独跑时会把 x/crypto 回退到 go-zero 要求的最低版本 v0.55.0，故需在引入后显式 `go get golang.org/x/crypto@v0.57.0` 固定。四项依赖现均为直接依赖（无 `// indirect`）。

### 5.6 逻辑层接口注入的必要调整（任务 2.11）

`model.UsersModel` 含未导出的 `withSession` 方法，接口无法在 `model` 包之外实现，测试也就无法注入替身。因此按「调用方定义自己需要的接口」在 `svc` 包新增 `UserRepository`（只含 logic 实际调用的四个方法）、`SessionStore`、`TokenIssuer` 三个接口，`ServiceContext` 的字段声明为接口类型。真实 model 仍可通过编译期断言 `_ UserRepository = model.UsersModel(nil)` 确认兼容。

