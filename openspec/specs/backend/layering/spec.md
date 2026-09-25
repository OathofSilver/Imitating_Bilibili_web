# backend/layering Specification

## Purpose
定义单体后端的分层模型与依赖方向，使业务域边界在文件系统上可见、业务规则集中在 service、数据访问集中在同域存储适配，避免业务逻辑漏进 HTTP 层或跨域直连对方数据。

## Requirements

### Requirement: 业务域按垂直切分组织

后端业务代码 SHALL 位于 `server/internal/<domain>/` 之下，按业务域垂直成包。域内文件 SHALL 按角色命名并保持扁平：`handler.go`、`service.go`、`repo.go`、`cache.go`、`search.go`、`model.go`、`errs.go`。当单个文件超过约 400 行时，SHALL 按子功能拆分文件（如 `handler_video.go`），MUST NOT 为此引入子目录。

MUST NOT 采用全局水平分层，即 MUST NOT 出现 `internal/handler/`、`internal/service/`、`internal/repo/` 这类按层而非按域组织的顶层目录。

#### Scenario: 新增一个业务域

- **WHEN** 开发者新增一个业务域
- **THEN** 该域 MUST 是 `server/internal/` 下的一个包，域内 MUST 按角色命名文件，MUST NOT 为其创建 `handler`/`service`/`repo` 子目录层

#### Scenario: 单个域文件过大

- **WHEN** 某个域内文件增长到需要拆分
- **THEN** 拆分 MUST 以同包内新增文件（如 `service_comment.go`）的方式完成，MUST NOT 改造成子目录

### Requirement: 依赖方向严格单向

依赖方向 SHALL 严格单向：`handler` 依赖 `service`，`service` 依赖同域的 `repo`、`cache`、`search`。`model` SHALL 作为叶子包，只承载领域模型与领域错误，MUST NOT 依赖任何上层包。

MUST NOT 出现反向依赖（`repo` 依赖 `service`）或跨层调用（`handler` 直接依赖 `repo`）。

#### Scenario: handler 需要读取数据

- **WHEN** 某个 HTTP 接口需要读取或写入数据
- **THEN** 调用链 MUST 经 `handler` 到 `service` 再到存储适配完成，`handler` MUST NOT 直接调用 `repo`

#### Scenario: 代码评审发现反向依赖

- **WHEN** 评审或静态检查发现下层包导入上层包
- **THEN** 该实现 MUST 判定为违规并重构以恢复单向依赖

### Requirement: 分层职责边界

`handler` SHALL 只负责参数绑定、参数校验、请求与响应 DTO 转换、调用 `service`、将错误映射为对外业务码。业务规则 MUST 位于 `service`。数据读写 MUST 通过同域的存储适配完成。

#### Scenario: 新增一个对外接口

- **WHEN** 开发者新增一个对外 HTTP 接口
- **THEN** 校验与 DTO 转换 MUST 在 `handler`，业务规则 MUST 在 `service`，数据访问 MUST 经同域存储适配，任一层职责越界 SHALL 判定为违规

#### Scenario: handler 中出现业务分支

- **WHEN** `handler` 中包含与具体接口无关的业务规则判断
- **THEN** 该规则 MUST 下沉到 `service`，MUST NOT 保留在 `handler`

### Requirement: 存储适配不可跨域访问

`repo`、`cache`、`search` SHALL 以未导出类型实现，只允许同域 `service` 访问。跨域访问他人数据 MUST 通过对方的 `service` 接口完成，MUST NOT 直接查询对方的表、缓存键或检索索引。

#### Scenario: A 域需要 B 域的数据

- **WHEN** 一个业务域需要另一个业务域的数据
- **THEN** 该数据 MUST 通过对方 `service` 的导出接口获取，MUST NOT 直接查询对方的表或缓存

### Requirement: 跨域数据传递必须经 DTO 转换

跨域调用返回的数据 SHALL 转换为本域定义的 DTO 后再使用，MUST NOT 直接复用其他域的领域模型或持久化结构体。领域模型 SHALL 只在本域内流转。

#### Scenario: 视频域需要作者昵称

- **WHEN** 视频域需要展示用户域的昵称
- **THEN** 用户域 `service` SHALL 返回其导出结构，视频域 MUST 将其转换为本域 DTO，MUST NOT 让用户域的领域模型穿透到视频域的 `handler`

### Requirement: service 接收 context 而非框架对象

`service` 层方法 SHALL 接收 `context.Context` 作为首个参数，MUST NOT 接收 `*gin.Context` 或其他 HTTP 框架对象。所有跨越 service 边界的调用 MUST 将 context 向下透传。

#### Scenario: 业务逻辑被非 HTTP 入口复用

- **WHEN** 同一段业务逻辑需要被消息消费者或定时任务调用
- **THEN** 该逻辑 MUST 能通过传入 `context.Context` 直接复用，MUST NOT 依赖 `*gin.Context` 而无法脱离 HTTP 层调用

#### Scenario: 存储调用缺少 context

- **WHEN** 代码调用数据库、缓存或检索服务时未传入 context
- **THEN** 该实现 MUST 判定为违规并补齐 context 透传

### Requirement: 事务边界唯一归属 service

数据库事务 SHALL 只在 `service` 层开启与提交。`repo` 层方法 MUST NOT 自行开启事务。一个业务流程 SHALL 对应一个事务边界，跨多个存储适配的一致性修改 MUST 在同一事务内完成，同一事务 MUST 只作用于单一数据库。

#### Scenario: 需要同时写多张表

- **WHEN** 一个业务流程需要原子地修改多张表
- **THEN** 事务 MUST 由 `service` 开启并覆盖全部写操作，存储适配 MUST 接收事务句柄而非自行开启

#### Scenario: 存储适配内部开启事务

- **WHEN** 评审发现 `repo` 层方法内部开启了自己的事务
- **THEN** 该实现 MUST 判定为违规，事务控制权 MUST 移回 `service`

### Requirement: 事务内禁止外部 I/O

在数据库事务的开启与提交之间，MUST NOT 执行数据库之外的网络调用，包括缓存读写、消息投递、检索写入与任何 HTTP 请求。

依赖异步或最终一致的能力 MUST 通过事务外投递机制实现，MUST NOT 以在事务内直接发起外部调用的方式实现。

#### Scenario: 写库同时需要发消息

- **WHEN** 一次数据变更需要同时产生一条异步消息
- **THEN** 消息 MUST NOT 在事务内直接投递，MUST 采用事务内落表加事务外投递的方式保证最终必达

#### Scenario: 事务内访问缓存

- **WHEN** 事务代码块中出现缓存读写调用
- **THEN** 该调用 MUST 移出事务，MUST NOT 以长事务持有连接等待网络返回

### Requirement: 平台层不含业务逻辑

跨域复用的基础设施 SHALL 位于 `server/internal/platform/` 之下，包含配置、统一响应、错误码表、中间件、存储客户端构造、日志与 ID 生成。平台层 MUST NOT 依赖任何业务域，MUST NOT 包含业务规则。

#### Scenario: 某能力被多个域共用

- **WHEN** 某项能力被多个业务域共用且不含业务规则
- **THEN** 该能力 MUST 位于 `platform/`，MUST NOT 复制到各业务域内

#### Scenario: 平台层引入业务概念

- **WHEN** `platform/` 下出现对具体业务实体或业务规则的引用
- **THEN** 该实现 MUST 判定为违规并将业务部分移回对应业务域

### Requirement: 配置集中注入且启动期校验

中间件连接信息与运行参数 SHALL 通过配置文件与环境变量注入，MUST NOT 硬编码在代码中。配置 SHALL 在进程启动时集中加载并校验，缺失必填项时 MUST 直接启动失败，MUST NOT 以默认值静默兜底。

#### Scenario: 接入新的中间件依赖

- **WHEN** 代码需要接入新的中间件
- **THEN** 其连接参数 MUST 来自配置，代码库中 MUST NOT 出现明文连接串或密钥

#### Scenario: 必填配置缺失

- **WHEN** 进程启动时必填配置项缺失
- **THEN** 进程 MUST 立即以非零状态退出并指明缺失项，MUST NOT 启动成功后在使用时才报错
