# Design

## Context

动机见 proposal.md。本设计面对三个已确认的现状约束：

1. **无任何数据访问代码**。`server/go.mod` 仅依赖 go-zero v1.10.3；`app/user/model/` 只有占位 README；五个域的 `internal/config/config.go` 都只有 `rest.RestConf` 与 `Domain`；`common/health` 的 mysql/redis 依赖项是用 TCP 端口探测（`probe`）伪造的，并非真实连接。
2. **认证中间件是空壳**。`common/middleware.Auth` 直接 `return next`，注释已声明由认证变更替换。
3. **凭证载体已有隐含指向**。`common/middleware.CORS` 的 `Allow-Headers` 已包含 `Authorization`，说明骨架预设凭证走请求头；但同一中间件设置 `Access-Control-Allow-Origin: *`，与 Cookie 方案不兼容。
4. **代码生成工具可用**。本机已安装 goctl v1.10.2，且已实测用本变更的建表语句执行 `goctl model mysql ddl` 可正常生成 model 层。因此骨架变更决策 3 预留的「goctl 不可用则手工等价结构降级」路径在本变更中不需要走。

同时 `establish-baseline-specs` 已对分层、目录、ID 生成、Redis 键规范形成硬约束，本变更是这些约束的第二次实际落地（第一次是骨架），因此完成后必须回头逐条核对。

## Goals / Non-Goals

**Goals:**

- 交付**可复用的存储基础设施**：MySQL 与 Redis 的连接能力放在 `common/`，而不是塞进 user 域，因为这层能力后续每个域都要用。
- 让 `identity/auth` 的「受保护接口服务端强制校验」第一次真正生效，并让五个域都具备校验能力。
- 以最小复杂度实现登录态撤销，使「退出登录」在服务端真实生效而非仅前端表演。
- 让本变更自身成为后续功能（视频、互动、评论）接入数据库与鉴权的模板。

**Non-Goals:**

- 不做单点登录、扫码登录、第三方登录、refresh token、验证码（见 proposal 的 Out of Scope）。
- 不引入 ORM 与迁移框架；本变更是项目第一次建表，用一份可重放的建表脚本即可。
- 不实现用户等级与权限的业务逻辑（`level`/`role` 仅落字段与默认值）。
- 不实现头像文件上传与存储。
- 不为其他四域新增业务接口；它们在本变更中只获得鉴权能力与 Redis 接入。

## Decisions

### 决策 1：基础设施首建独立为第一阶段，先于一切鉴权业务代码

第一阶段交付 `common/` 的连接构造、五域配置扩展、user 域 model 层与 `users` 建表脚本，并把 health 的 mysql/redis 项从端口探测升级为真实连接校验；第二阶段才写注册登录。

- **理由**：这层地基是本变更中唯一「一旦定错、后面每个功能都要跟着错」的部分。把它前置并单独验收，可以让注册登录的实现只关心业务，不会顺手在 logic 里塞连接代码。
- **备选方案**：边写登录边建连接 → 否决。连接的生命周期管理最容易在赶进度时被打散到各处，之后每次新功能都要重复面对。

### 决策 2：凭证采用纯 JWT 自校验，配合 Redis 会话版本号实现撤销

JWT（HS256）载荷包含 `uid`、`ver`（会话版本号）、`iat`、`exp`，有效期 7 天。校验流程为：本地验签与有效期校验 → 读取 Redis 中该用户的当前版本号 → 版本不匹配则拒绝。登录成功时递增版本号并写入凭证；退出登录时再递增一次，使所有既有凭证立即失效。

Redis 键形如 `bw:v1:auth:ver:<uid>`，读写时设置 TTL 为凭证有效期的两倍以上（键自然过期时，该用户不可能还持有未过期凭证，语义自洽）。读取到空值视为版本 0；若 Redis 中数据意外丢失导致版本回退，凭证会被拒绝——宁可 fail-closed。

- **理由**：纯无状态 JWT 无法撤销；只要引入撤销，就必须在一次请求中查一次共享存储。版本号方案比「黑名单 + jti」键数量少一个量级、无需按设备管理，也比「记录撤销时间戳 + 比 iat」更严谨——后者在同一秒内完成登录与退出时存在精度漏洞。
- **备选方案**：
  - 纯无状态、不做撤销 → 否决。退出登录将只是前端清除本地存储，安全语义不成立，spec 中的「登录态撤销」无从实现。
  - 撤销名单按 jti 存储 → 否决。键数量随登录次数线性增长，且其价值（踢单个设备）属于多端管理，已明确不做。
  - 其他域通过 RPC 调用用户域校验 → 否决。会让每个受保护请求多一次跨服务跳转，并让其他域的可用性强依赖用户域。

### 决策 3：鉴权能力下沉 `common/`，五个域全部接入 Redis

`common/middleware.Auth` 接受一个「凭证撤销检查器」接口，user 域注入基于 Redis 的实现；同时 Redis 客户端构造能力放 `common/`，五个域在配置中各接入一次。

- **理由**：`backend/architecture` 明确要求「跨服务复用的公共能力（响应封装、错误码、ID 生成、中间件）SHALL 位于 `common/`，MUST NOT 复制到各域目录内」。且四个非用户域后续必然要用 Redis（视频计数、评论缓存），本变更一次性接入不构成浪费。
- **代价**：所有域的受保护请求都会多一次 Redis 读取，且各域可用性开始依赖 Redis。缓解见 Risks。
- **备选方案**：只在 user 域接入 Redis，其他四域暂不校验撤销 → 否决。会与本变更的 spec 直接冲突（spec 要求撤销后访问任意受保护接口都被拒绝）。

### 决策 4：MySQL 统一走 go-zero `core/stores/sqlx`，model 层由 goctl 生成后按软删除约定定制

数据访问层使用 go-zero 的 `core/stores/sqlx`（不是裸用 `database/sql`，也不是社区库 `github.com/jmoiron/sqlx`），驱动为 `github.com/go-sql-driver/mysql`。model 层骨架由 goctl 生成，产物落 `server/app/user/model/`。

**sqlx 侧（已直接核对 go-zero v1.10.3 源码）：**

- 连接构造用 `sqlx.NewConn(sqlx.SqlConf{DataSource, DriverName})`，返回 `(SqlConn, error)`。不使用 `MustNewConn` 与 `NewMysql`——二者失败时走 `logx.Must` 直接终止进程，与 `backend/architecture`「错误逐层包装且不得吞掉」不符。`SqlConf` 另含 `Replicas` 与 `Policy`，本变更单实例部署不启用。
- `QueryRow` / `QueryRows` 通过 `db` 结构体标签自动完成列到字段的映射，无需手写 `rows.Scan` 循环；`QueryRowPartial` / `QueryRowsPartial` 为非严格模式。
- `QueryRow` / `QueryRows` 是严格模式：结果集中出现无对应字段的列即报错。goctl 生成的字段列表来自 `builder.RawFieldNames`，天然逐列显式，不产生 `SELECT *`；该约束对手写查询同样适用。
- v1.10.3 未提供重复键判定辅助函数（全仓检索无 `IsDuplicate`）。用户名冲突需自行判定 MySQL 错误码 1062：`errors.As(err, &mysqlErr)` 后比较 `mysqlErr.Number`，故 `github.com/go-sql-driver/mysql` 是**直接依赖**而非仅传递依赖。sqlx 内置熔断器已把 1062 登记为 acceptable，重复键错误不会触发熔断。

**goctl 侧（已实测 v1.10.2 的生成行为）：**

- 生成命令：`goctl model mysql ddl -src deploy/users.sql -dir app/user/model`（包名取目标目录名，落在 `model` 包）。**不传 `-c`**：无缓存版本的构造签名是 `NewUsersModel(conn sqlx.SqlConn)`，而 goctl 的缓存键默认前缀为 `cache`，与 `data/platform` 的「业务前缀 + 结构版本」规范不符，且 user 表在本变更无缓存需求。
- 生成三个文件：`usersmodel_gen.go`（声明 `DO NOT EDIT`，含结构体、字段列表与 CRUD）、`usersmodel.go`（明示「add more methods here」的定制入口）、`vars.go`（`ErrNotFound` 别名）。`deleted_at` 正确映射为 `sql.NullTime`，并已由唯一索引自动生成 `FindOneByUsername`。
- **四处生成代码与基线规范冲突，必须在 `usersmodel.go` 的 `customUsersModel` 上覆盖**：
  1. `Delete(ctx, id)` 生成的是物理 `delete from users where id = ?`，直接违反 `data/platform` 的软删除要求。**覆盖为软删除**（写 `deleted_at`）而非弃用——弃用会留下一个随时可能被误调的物理删除入口。
  2. `FindOne` 与 `FindOneByUsername` 不带 `deleted_at IS NULL`，会返回已软删除用户。**覆盖为过滤软删除**，否则 `identity/profile` 的「目标用户已被删除时返回 40400」无法成立。
  3. `Update(ctx, data)` 生成的是「更新除 `id` 与时间字段外的全部列」，**包含 `username`**。用它改资料会写坏用户名，违反 `identity/profile` 的不可变字段要求。资料修改走自定义的白名单方法，不使用生成的 `Update`。
  4. 生成时输出的两条 warning 属预期，**不得据此改表**：主键建议加 `AUTO_INCREMENT`（`data/platform` 要求对外 ID 用 Snowflake，不加自增才是正确形态）、部分列建议加 `DEFAULT`（`username` 与 `password_hash` 不该有默认值）。
- 再生成的幂等性已实测：重复执行生成命令时 `usersmodel_gen.go` 被重新生成，而 `usersmodel.go` 输出「already exists, ignored」并原样保留。**故定制只能写在 `usersmodel.go`**，写进 `_gen.go` 的内容会在下次生成时丢失。

- **理由**：goctl 免去了结构体与字段列表的样板维护，而它不理解的软删除与字段白名单语义恰好只有四处需要覆盖，且集中在同一个定制文件内，可随生成命令反复重建。
- **备选方案**：纯手写 model 层 → 否决，需自行维护字段列表与结构映射，收益不抵成本；启用 goctl 缓存（`-c`）→ 否决，用户表无缓存需求，且默认键前缀与 Redis 键规范冲突；`github.com/jmoiron/sqlx` → 否决，能力与 go-zero 自带版本高度重叠却多一套依赖，在 go-zero 项目里属少见组合；GORM → 否决，与已锁定的分层与手写 SQL 风格冲突，且会隐藏慢查询。

### 决策 5：接口路径用 `/me` 承担「本人」，避免路径匹配歧义

| 方法 | 路径 | 登录要求 |
| --- | --- | --- |
| POST | `/api/v1/user/register` | 否 |
| POST | `/api/v1/user/login` | 否 |
| POST | `/api/v1/user/logout` | 是 |
| GET | `/api/v1/user/me` | 是 |
| PUT | `/api/v1/user/me` | 是 |
| GET | `/api/v1/user/users/{id}` | 否 |

- **理由**：`/profile` 与 `/profile/{id}` 这种「同前缀 + 可选路径参数」的组合在路由注册顺序上容易踩坑，用 `/me` 与 `/users/{id}` 前缀完全分离，零歧义；且「修改本人」由凭证身份决定目标，接口不接收目标 ID，天然杜绝越权改他人。
- **备选方案**：`/api/v1/user/profile` + `/api/v1/user/profile/{id}` → 被否，理由同上。

### 决策 6：新增两个错误码，其余复用既有区段

复用 `40001`（参数非法）、`40100`（未认证）、`40400`（资源不存在）；新增 `40901`（用户名已存在）。

- **理由**：`api/contract` 要求错误码集中定义、按区段划分且禁止魔法数字。用户名冲突属于客户端侧错误，放在 `4xxxx` 段的 `40901` 位置语义明确。
- **备选方案**：复用 `40001` 表达用户名冲突 → 否决。让前端无法仅凭 code 区分「字段格式错」与「用户名被占用」，违反 `api/contract` 的「前端 MUST 能仅凭 code 区分错误类型」。

### 决策 7：凭证存 localStorage 并走 `Authorization` 请求头

前端在请求层统一读取本地凭证并注入 `Authorization: Bearer <token>`；收到 `40100` 时清除凭证并跳转登录。

- **理由**：CORS 中间件已预留 `Authorization` 头，且当前 `Access-Control-Allow-Origin: *` 与 Cookie 凭证互斥——改用 Cookie 需要同时收紧 CORS、引入 `credentials` 配置并新增 CSRF 防护，属于独立工作量。
- **代价**：localStorage 对 XSS 无抵抗力。缓解见 Risks。
- **备选方案**：httpOnly Cookie → 保留为后续安全加固方向，不在本变更内。

### 决策 8：只做单凭证，不做 refresh token

签发单一凭证，有效期 7 天；过期后用户重新登录。

- **理由**：refresh token 的价值在于「短 access 寿命 + 免打扰续期」，而本项目已用 Redis 版本号覆盖了撤销需求，引入双凭证会把前端请求层的重试与并发刷新逻辑复杂度抬高一个层级，与「简单鉴权」的目标相悖。
- **备选方案**：access + refresh 双凭证 → 保留为后续演进方向；届时 `identity/auth` 的「登录态签发与携带」需要以 delta 修订。

## Risks / Trade-offs

- **每个受保护请求多一次 Redis 读取** → 缓解：读取是单键 GET，成本可控；Redis 与各域同机部署。若后续成为瓶颈，可在各域加秒级本地缓存「版本号未变更」的结论，本变更不实现。
- **Redis 不可用会导致全部受保护接口不可用（fail-closed）** → 缓解：这是刻意的安全选择，错误码使用既有的 `50001` 依赖不可用并明确提示，而非静默放行；health 接口会先行暴露 Redis 状态。
- **服务端版本号数据丢失会把用户登出** → 缓解：可接受，安全侧失败；重新登录即可恢复。
- **localStorage 凭证暴露于 XSS** → 缓解：项目不渲染用户提供的 HTML；后续可迁移至 httpOnly Cookie。
- **go-zero sqlx 内置熔断器在高错误率下会打开并返回服务不可用** → 缓解：属框架自带的故障保护，非缺陷；错误仍映射为统一错误码 `50001` 并逐层包装记录，便于区分「熔断打开」与「真实故障」。
- **`QueryRow` 严格模式对列与字段的对应关系要求严格** → 缓解：goctl 生成与手写查询一律显式列出字段、不用 `SELECT *`，故给 `users` 表加列不会影响既有查询；该约束写入任务 1.6 的验收项。
- **定制内容若写进 `usersmodel_gen.go` 会在下次生成时丢失** → 缓解：所有定制一律写在 `usersmodel.go`（已实测重复生成时该文件被跳过）；任务 1.6 的验收含「重新执行生成命令后自定义方法仍在」。
- **goctl 生成的 CRUD 语义与基线规范有四处冲突，容易被直接沿用** → 缓解：冲突点与覆盖要求在决策 4 中逐条列明，并作为任务 1.6 的显式交付内容；物理删除的 `Delete` 被覆盖而非弃用，以消除误调入口。
- **重复键判定依赖 MySQL 错误码 1062 这一具体驱动行为** → 缓解：判定逻辑收敛在 model 层单点，更换驱动只需改一处；并编写单元测试覆盖该分支。
- **五域同时接入 Redis 使配置变多** → 缓解：配置项与构造调用在 `common/` 统一定义，各域只有三行配置与一次构造。
- **bcrypt 的哈希计算是 CPU 密集操作** → 缓解：注册/登录是低频接口，可接受；不将 bcrypt 用于任何高频路径。
- **建表脚本缺乏迁移框架，后续表结构变更无版本记录** → 缓解：脚本以 `deploy/` 下的可重放 SQL 形式存在，后续引入正式迁移工具时以其为初始版本。

## Migration Plan

1. 先跑 `make up` 确认 MySQL 与 Redis 容器可达。
2. 执行 `server/deploy/` 下的建表脚本创建 `users` 表（脚本需可重复执行，使用 `CREATE TABLE IF NOT EXISTS`）。
3. 各域配置补入存储与密钥项，通过环境变量覆盖默认值；开发环境密钥可用占位值，但 MUST NOT 提交真实生产密钥。
4. 分两阶段提交：第一阶段只含基础设施与 health 升级（此时无任何业务接口变化，可单独验证）；第二阶段加入鉴权业务接口与前端。
5. 回滚：第一阶段回滚只需还原配置与 `common/` 新增文件，`users` 表删除即可（此时无业务数据）；第二阶段回滚需同时回滚前端凭证注入，否则登录页会持续报错。

## Open Questions

- 用户等级 `level` 的成长规则（经验值来源、升级阈值）尚未定义。本变更只落字段与默认值 0，规则属于后续独立变更，不影响本变更的 spec 与任务拆分。
- 管理员角色 `role` 对应的具体权限点尚未定义。本变更只落字段与默认值 0，与 `identity/auth` 已确立的权限模型基线一致。

## 触及的基线规范逐条核对

按 `openspec/config.yaml` 的 `rules.proposal` 要求，对本变更触及的基线规范逐条核对：

| 基线能力 | 相关条目 | 核对结论 |
| --- | --- | --- |
| `identity/auth` | 登录态签发与携带 | 符合。JWT 由服务端签发、带 `exp` 有效期，前端在 `api/http.ts` 统一拦截位置注入。 |
| `identity/auth` | 凭证过期统一处理 | 符合。服务端对过期/无效凭证统一返回 `40100`；前端接到该码清除凭证并跳转登录，保留原目标地址。 |
| `identity/auth` | 受保护接口服务端强制校验 | 符合。`middleware.Auth` 由空壳替换为真实校验；前端守卫仅作体验优化。 |
| `identity/auth` | 用户等级与权限模型 | 符合。`level`/`role` 字段落库；修改他人资料的越权路径由「接口不接受目标 ID」在设计上消除，并在 spec 中固化为场景。 |
| `identity/auth` | 敏感信息不得外泄 | 符合。响应 DTO 不含 `password_hash`；他人公开资料不返回联系方式，已在 spec 中写成场景。 |
| `identity/profile` | 全部（新增能力） | 新增，无既有约束需要满足；其 requirement 已覆盖字段白名单、可见性差异与默认值。 |
| `api/contract` | 统一响应结构 | 符合。复用 `common/response`，HTTP 状态码仅表达传输层语义。 |
| `api/contract` | 路径版本前缀 | 符合。六个接口全部位于 `/api/v1/user/` 下。 |
| `api/contract` | 集中定义的错误码体系 | 符合。仅新增 `40901` 并集中登记于 `common/errcode`，无魔法数字。 |
| `api/contract` | 统一分页约定 | 不适用。本变更无列表接口。 |
| `api/contract` | 写操作幂等 | 不适用。注册以用户名唯一约束天然防重；资料修改是覆盖写，重复提交结果一致。 |
| `api/contract` | 参数校验与错误信息 | 符合。校验在 `api` 层完成，失败返回 `40001` 并在 message 中指明字段名。 |
| `backend/architecture` | 三层职责与单向依赖 | 符合。handler 只做绑定与响应，业务规则在 logic 层，SQL 只在 model 层。 |
| `backend/architecture` | 仓库目录布局与单模块组织 | 符合。新增文件全部落在 `server/common/`、`server/app/user/{api,model}/`、`server/deploy/`；连接构造放在 `common/` 而非某域内。 |
| `backend/architecture` | 对外路径按业务域划分 | 符合。全部接口位于 `/api/v1/user/`。 |
| `backend/architecture` | 跨服务调用必须走 RPC | 不适用。本变更不跨域取数据；凭证校验走共享密钥与共享 Redis，不属于「取另一业务域的数据」。 |
| `backend/architecture` | 服务按业务域划分 | 符合。鉴权能力归入用户域，未新建域。 |
| `backend/architecture` | 配置外部化与环境隔离 | 符合。数据库、Redis、密钥全部走 `etc/*.yaml` 与环境变量，代码内无明文连接串与密钥。 |
| `backend/architecture` | 错误逐层包装且不得吞掉 | 符合。model 层错误向上包装，对外映射为统一错误码，不暴露内部细节。 |
| `data/platform` | 全局 ID 由 Snowflake 生成 | 符合。`users.id` 由既有 `common/snowflake` 生成，非自增。 |
| `data/platform` | MySQL 建模与软删除约定 | 符合，但需覆盖生成代码。表含 `created_at`/`updated_at`/`deleted_at`；goctl 生成的 `Delete` 是物理删除、`FindOne*` 不带软删除过滤，二者已在 `customUsersModel` 中覆盖（决策 4）。 |
| `data/platform` | Redis 缓存读写与键规范 | 符合。会话版本号键带业务前缀 `bw`、结构版本 `v1`、业务段 `auth`，且强制设置 TTL；此外刻意不启用 goctl 的缓存生成，避免其默认 `cache` 前缀与版本标识要求冲突。 |
| `data/platform` | 计数类数据走 Redis 累加与异步落库 | 不适用。本变更无计数场景。 |
| `data/platform` | 消息可靠投递与幂等消费 | 不适用。本变更不使用消息队列。 |
| `data/platform` | Elasticsearch 检索与降级 | 不适用。本变更不涉及检索。 |
| `frontend/conventions` | 目录结构约定 | 符合。新增文件落在 `pages`、`features`、`store`、`api`、`types`，不新增顶层目录。 |
| `frontend/conventions` | 组件实现约定 | 符合。表单与资料展示组件置于 `features`，通用组件不含接口调用。 |
| `frontend/conventions` | 状态管理边界 | 符合。仅登录态（凭证与当前用户）进入 Zustand，因其被多页面共享；资料列表等数据不入全局 store。 |
| `frontend/conventions` | 路由与访问守卫 | 符合。守卫在 `app` 集中声明，未登录跳转登录并保留原目标地址；权限判定仍在服务端。 |
| `frontend/conventions` | 主题与样式约束 | 符合。表单与按钮样式使用 Mantine 主题 token，无硬编码色值。 |
| `frontend/conventions` | 类型与接口封装 | 符合。请求与响应类型定义于 `types/` 并在 `api/` 封装，页面不直接调用 fetch。 |
| `quality/testing` | 全部 | 符合。每个任务在 tasks.md 中携带可验证的验收步骤；后端逻辑以接口注入依赖以便测试。 |

**核对结论：无违反项，无需 MODIFIED delta。**
