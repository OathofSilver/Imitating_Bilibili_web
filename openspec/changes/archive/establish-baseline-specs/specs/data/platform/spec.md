# Spec Delta

## Purpose

定义全局 ID 生成、MySQL 建模与删除策略、Redis 缓存与计数、RabbitMQ 异步边界以及 Elasticsearch 检索一致性的统一规则，防止各模块各自选择存储与同步方式导致数据不一致。

## ADDED Requirements

### Requirement: 全局 ID 由 Snowflake 生成

所有对外暴露的业务实体 ID SHALL 由 Snowflake 算法生成。数据库自增主键 MUST NOT 作为对外 ID 使用；自增主键 MAY 仅作为内部物理主键存在。

#### Scenario: 新增一张业务表

- **WHEN** 开发者新增业务表并需要对外暴露 ID
- **THEN** 该 ID MUST 由 Snowflake 生成，接口响应中 MUST NOT 出现自增字段值

### Requirement: MySQL 建模与软删除约定

业务表 SHALL 包含创建时间与更新时间字段；业务实体的删除 SHALL 采用软删除（标记删除时间），MUST NOT 物理删除业务数据。跨表一致性修改 SHALL 在同一事务内完成。

#### Scenario: 用户删除自己发布的评论

- **WHEN** 用户执行删除评论操作
- **THEN** 该评论 MUST 被标记为已删除而非从库中移除，其关联计数 MUST 在同一流程中修正

### Requirement: Redis 缓存读写与键规范

读路径 SHALL 采用 cache-aside 模式：先读缓存，未命中则回源并写回缓存。所有缓存键 MUST 带业务前缀与结构版本标识，且 MUST 设置过期时间。数据更新时 SHALL 主动失效对应缓存。

#### Scenario: 缓存未命中

- **WHEN** 查询请求未命中缓存
- **THEN** 系统 MUST 回源数据库并将结果写回缓存，且写入的键 MUST 带业务前缀与版本号

#### Scenario: 业务数据被更新

- **WHEN** 业务数据发生写变更
- **THEN** 对应缓存 MUST 被失效或更新，MUST NOT 留下与数据库不一致的长期脏数据

### Requirement: 计数类数据走 Redis 累加与异步落库

高频计数（播放量、点赞数、投币数、收藏数、评论数、弹幕数）SHALL 在 Redis 中累加，并通过消息队列异步落库到 MySQL，MUST NOT 在请求路径上同步执行计数写库。

#### Scenario: 播放量上报

- **WHEN** 客户端上报一次播放
- **THEN** 计数 MUST 在 Redis 中累加，落库 MUST 通过消息队列异步完成，请求 MUST NOT 因写库而阻塞

### Requirement: 消息可靠投递与幂等消费

发送到消息队列的消息 SHALL 持久化；消费端 SHALL 具备幂等能力，按业务键去重，重复投递 MUST NOT 产生重复副作用。消费失败 SHALL 进入重试或死信处理，MUST NOT 静默丢弃。

#### Scenario: 同一条消息被重复投递

- **WHEN** 消费端收到重复消息
- **THEN** 消费端 MUST 依据业务键识别重复并跳过重复处理，数据 MUST 保持与处理一次一致

### Requirement: Elasticsearch 检索与降级

MySQL SHALL 作为权威数据源，Elasticsearch 索引 SHALL 通过异步消息同步。搜索功能 MUST 具备降级能力：检索服务不可用时 SHALL 返回降级结果或明确的不可用提示，MUST NOT 直接向上抛出内部错误导致整体不可用。

#### Scenario: 检索服务不可用

- **WHEN** 搜索请求到达而 Elasticsearch 不可用
- **THEN** 系统 MUST 走降级路径返回可用结果或明确错误码，MUST NOT 返回未处理的 5xx

#### Scenario: 视频信息更新后检索结果

- **WHEN** 视频标题在 MySQL 中被修改
- **THEN** 索引 MUST 通过异步消息在有限时间内更新，搜索结果最终 MUST 与数据源一致
