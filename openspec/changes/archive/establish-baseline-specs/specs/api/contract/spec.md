# Spec Delta

## Purpose

定义前后端之间 HTTP 接口的统一契约，包括响应结构、路径版本、错误码、分页与幂等规则，使前端可用一套解析逻辑处理全部响应，避免各接口自造格式。

## ADDED Requirements

### Requirement: 统一响应结构

所有 HTTP 接口 SHALL 返回统一结构 `{ code, message, data }`。业务失败 MUST 使用该结构中的业务错误码表达，MUST NOT 依赖 HTTP 状态码向前端传递业务语义。HTTP 状态码 SHALL 仅表达传输层语义（如 401 未认证、5xx 服务不可用）。

#### Scenario: 接口调用成功

- **WHEN** 接口业务处理成功
- **THEN** 响应 `code` MUST 为 0，`data` MUST 承载业务数据

#### Scenario: 接口业务失败

- **WHEN** 接口业务处理失败（如资源不存在、参数非法、无权限）
- **THEN** 响应 MUST 携带对应的非 0 业务 `code` 与可读 `message`，前端 MUST 能仅凭 `code` 区分错误类型

### Requirement: 路径版本前缀

所有对外接口路径 SHALL 以 `/api/v1` 前缀开头。破坏性变更 MUST 通过新增版本前缀引入，MUST NOT 在既有版本上修改响应语义。

#### Scenario: 新增一个接口

- **WHEN** 开发者新增对外接口
- **THEN** 其路径 MUST 位于 `/api/v1` 之下

#### Scenario: 需要修改既有响应的字段语义

- **WHEN** 某接口返回结构需要发生不兼容变更
- **THEN** 该变更 MUST 以新版本路径提供，旧版本 MUST 在过渡期内保持原语义

### Requirement: 集中定义的错误码体系

错误码 SHALL 集中定义并按区段划分：`0` 表示成功，`4xxxx` 段表示客户端侧错误（参数、鉴权、权限、资源不存在），`5xxxx` 段表示服务端侧错误。新增错误 MUST 在集中定义处登记，MUST NOT 在业务代码中散落魔法数字。

#### Scenario: 新增一种错误类型

- **WHEN** 业务需要返回新的错误类型
- **THEN** 该错误码 MUST 在集中错误码表中登记并归属正确区段

### Requirement: 统一分页约定

列表类接口 SHALL 使用 `page` 与 `page_size` 作为分页入参，并在响应中返回 `has_more` 指示是否还有下一页。总数 `total` MAY 在可精确统计时返回，无法精确统计时 MUST 省略而非返回估算值。

#### Scenario: 请求页码超出数据范围

- **WHEN** 客户端请求的 `page` 超出实际数据范围
- **THEN** 接口 MUST 返回空列表且 `has_more` 为 false，MUST NOT 返回错误

#### Scenario: 分页参数非法

- **WHEN** `page` 或 `page_size` 非法（非正数或超出上限）
- **THEN** 接口 MUST 返回参数非法错误码，MUST NOT 静默使用默认值

### Requirement: 写操作幂等

会产生副作用的写接口（点赞、投币、收藏、关注等）SHALL 具备幂等性：同一用户对同一目标的重复提交 MUST 不产生重复副作用，且重复调用 MUST 返回一致的最终状态。

#### Scenario: 用户重复点赞同一视频

- **WHEN** 同一用户连续多次提交对同一视频的点赞请求
- **THEN** 点赞记录 MUST 只存在一条，计数 MUST NOT 重复累加，响应 MUST 反映最终已点赞状态

### Requirement: 参数校验与错误信息

参数校验 SHALL 在 `api` 层完成；校验失败 MUST 返回参数非法错误码并指明具体字段，MUST NOT 将未校验参数透传至业务层。

#### Scenario: 必填字段缺失

- **WHEN** 请求缺少必填字段或字段格式非法
- **THEN** 接口 MUST 返回参数非法错误码，并在 `message` 中指明字段名
