# Backend Architecture Spec
## Purpose

定义后端服务的分层结构、依赖方向与调用边界，确保参数校验集中在网关层、业务逻辑集中在 rpc 层、数据访问集中在 model 层，避免业务逻辑散落到网关或直接嵌入数据访问代码。

## Requirements
### Requirement: 三层职责与单向依赖

后端 SHALL 按 `api`、`rpc`、`model` 三层组织，依赖方向 MUST 严格单向：`api` → `rpc` → `model`。上层 MAY 依赖下层，下层 MUST NOT 反向依赖上层，同层之间 MUST NOT 横向直连数据库以外的私有实现。

- `api` 层 SHALL 只负责路由、参数绑定与校验、响应封装。
- `rpc` 层 SHALL 承载全部业务规则与事务编排。
- `model` 层 SHALL 只负责数据访问与存储结构映射。

#### Scenario: 新增一个业务接口时的分层落位

- **WHEN** 开发者新增一个对外 HTTP 接口
- **THEN** 参数校验与响应封装 MUST 位于 `api` 层，业务规则 MUST 位于 `rpc` 层，数据读写 MUST 通过 `model` 层完成，任一层缺失或职责越界 SHALL 判定为违反本规范

#### Scenario: 网关层直接编写业务 SQL

- **WHEN** 代码评审或静态检查发现 `api` 层直接执行业务 SQL 或编排事务
- **THEN** 该实现 MUST 被判定为违规并重构为 `api` 调用 `rpc`

### Requirement: 仓库目录布局与单模块组织

后端代码 SHALL 位于仓库的 `server/` 目录，并 SHALL 使用单一 Go module（模块根位于 `server/`）。业务域服务 SHALL 位于 `app/<domain>/` 下，每个域 SHALL 包含 `api`、`rpc`、`model` 三个子目录；跨服务复用的公共能力（响应封装、错误码、ID 生成、中间件）SHALL 位于 `common/`；部署与数据库脚本 SHALL 位于 `deploy/`。

#### Scenario: 新增一个业务域服务

- **WHEN** 开发者新增一个业务域
- **THEN** 该域 MUST 位于 `app/<domain>/` 且包含 `api`、`rpc`、`model` 三个子目录，MUST NOT 为该域单独创建 Go module

#### Scenario: 新增跨服务公共能力

- **WHEN** 某个能力被多个域共用
- **THEN** 该能力 MUST 位于 `common/`，MUST NOT 复制到各域目录内

### Requirement: 对外路径按业务域划分

对外 HTTP 路径 SHALL 形如 `/api/v1/<domain>/<resource>`，由网关按域前缀转发到对应服务的 `api` 层。系统 MUST NOT 引入额外的 BFF 聚合层：需要跨域数据的页面 SHALL 由前端并发调用多个域接口，或由对应域服务通过 RPC 聚合后返回。

#### Scenario: 新增一个视频相关接口

- **WHEN** 开发者新增视频域的对外接口
- **THEN** 其路径 MUST 位于 `/api/v1/video/` 之下，MUST NOT 挂在无域前缀的路径上

#### Scenario: 页面需要同时展示视频与作者信息

- **WHEN** 一个页面需要视频数据与作者数据
- **THEN** 该数据 SHALL 由视频域服务通过 RPC 聚合后一次返回，或由前端分别调用 `/api/v1/video/` 与 `/api/v1/user/`，MUST NOT 为此新建聚合层服务

### Requirement: 跨服务调用必须走 RPC

一个业务域需要另一个业务域的数据或能力时，SHALL 通过对应的 RPC 接口调用，MUST NOT 直连对方数据库表或复用对方 `model` 层代码。

#### Scenario: 视频服务需要用户昵称

- **WHEN** 视频详情接口需要展示作者昵称
- **THEN** 视频服务 MUST 调用用户服务的 RPC 接口获取，MUST NOT 直接查询用户库的用户表

### Requirement: 服务按业务域划分

后端 SHALL 按业务域拆分为独立服务，首批域至少包含：用户域、视频域、互动域（点赞/投币/收藏）、评论域、搜索域。新增业务能力 MUST 归入既有业务域；只有在既有域均无法承载时 MAY 新建域，且 MUST 在本规范中以 delta 形式登记。

#### Scenario: 新增弹幕能力

- **WHEN** 需要新增弹幕功能
- **THEN** 该能力 MUST 归入视频域或经 delta 登记的新域，MUST NOT 以独立散落模块形式实现

### Requirement: 配置外部化与环境隔离

数据库、Redis、RabbitMQ、Elasticsearch 等中间件连接信息 SHALL 通过配置文件或环境变量注入，MUST NOT 硬编码在代码中；不同环境 MUST 使用独立配置。

#### Scenario: 新增一个中间件依赖

- **WHEN** 代码需要接入新的中间件
- **THEN** 连接参数 MUST 来自配置，代码库中 MUST NOT 出现明文连接串或密钥

### Requirement: 错误逐层包装且不得吞掉

错误 SHALL 沿调用链逐层包装并携带上下文，MUST NOT 被静默丢弃；对外响应 MUST 映射为统一错误码，不得将内部错误细节直接暴露给客户端。

#### Scenario: 下游服务返回错误

- **WHEN** rpc 层调用下游依赖失败
- **THEN** 该错误 MUST 被包装上下文后向上传递并记录日志，MUST NOT 以空结果或忽略方式处理
