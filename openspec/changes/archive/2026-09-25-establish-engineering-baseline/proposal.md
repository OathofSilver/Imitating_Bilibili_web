# Proposal

## Why

项目已重构为空仓，技术栈（Go + Gin 单体 / React 18 + TypeScript + Vite 7 + Mantine + Zustand + React Router v7 / MySQL + Redis + RabbitMQ + Elasticsearch + IK / go-cache 本地缓存）与仓库定位（单体部署的类 B 站 Web 端全栈项目）已经确定，但这些约定只存在于讨论与零散笔记中，没有任何可校验、可追踪的载体。

后果是：第一个功能变更开始时，分层方式、ID 序列化、错误码分段、缓存一致性策略、事务边界这些决策都要重新拍一次，且每次可能拍出不同结果。更要紧的是，这些决策一旦在代码落地后才改变，代价就是重构。

现在做是因为代码量仍为零——此时固化约束成本最低，一旦骨架落地再回头统一就要付出返工代价。本变更把这八项工程约束固化为能力规范，作为后续所有功能变更的约束源与评审依据。

## What Changes

建立八项基线能力规范，均为新增，无既有规范被修改。

需要明确：这些规范描述的是**工程约束**而非运行时功能。其验收方式是代码结构检查、静态分析、CI 门禁与代码评审，而不是运行时功能测试——后续变更不应为它们补写功能测试。

- `backend/layering`：垂直切域的单体分层模型、严格单向依赖、事务边界归属、跨域调用规则、目录布局与文件组织。
- `backend/coding`：命名与注释语言约定、错误逐层包装、结构化日志、context 传递、goroutine 生命周期、禁止直接序列化持久化实体。
- `api/contract`：统一响应体、HTTP 状态码与业务码的双层职责划分、错误码分段、对外 ID 字符串化、游标分页、写操作幂等、限流响应。
- `data/platform`：MySQL 建模与迁移约定、Redis 键规范与缓存一致性策略、go-cache 本地缓存边界与跨实例失效、RabbitMQ 可靠投递与幂等消费、Elasticsearch + IK 索引设计与降级。
- `web/conventions`：前端目录结构、组件与状态管理边界、路由守卫、Mantine 主题与样式约束、类型与接口封装。
- `quality/testing`：测试分层、覆盖率门禁、集成测试隔离方式、异步最终一致性断言方式、缺陷回归用例要求。
- `delivery/git`：分支模型与命名对齐关系、提交信息规范、PR 模板要素、禁止提交项、换行符策略。
- `delivery/ci`：CI 流水线各阶段门禁，包含规范校验环节。

同时补充项目级上下文，这部分不属于能力规范：

- 更新 `openspec/config.yaml` 的 `context`：写入精简的项目定位、技术栈与三条不可协商的铁律。
- 更新 `openspec/config.yaml` 的 `rules`：要求提案必须列出所触及的基线能力。
- 新增 `openspec/project.md`：仓库定位，含「是什么 / 给谁用 / 解决什么 / Non-goals」。

## Capabilities

### New Capabilities

- `backend/layering`: 后端分层模型与依赖方向约束。含垂直切域的域包组织、handler 到 service 到存储适配的单向依赖、事务边界唯一归属、事务内禁止外部 I/O、跨域调用只经 service 接口、跨域数据传递必须经 DTO 转换。
- `backend/coding`: 后端编码规范。含标识符与注释的语言分工、错误包装与哨兵错误定义位置、结构化日志与请求标识、context 传递强制、goroutine 退出路径要求、常量收敛、小接口原则、禁止直接序列化持久化实体。
- `api/contract`: 前后端 HTTP 接口契约。含统一响应体结构、HTTP 状态码只表达传输层而业务语义走业务码、错误码分段登记、对外 ID 字符串化、游标分页为默认、写操作幂等、限流响应约定、参数校验位置。
- `data/platform`: 数据层与中间件使用策略。含 MySQL 建模与迁移与索引约定、Redis 键命名与 TTL 与缓存一致性、go-cache 可缓存数据边界与跨实例失效通道、RabbitMQ 命名与可靠投递与幂等消费与 Outbox、Elasticsearch 索引别名与显式 mapping 与 IK 分词与深分页与降级。
- `web/conventions`: 前端工程约定。含目录结构落位、组件实现与复用边界、Zustand 状态边界、路由集中声明与登录守卫、主题 token 与样式约束、类型定义与接口封装位置。
- `quality/testing`: 测试与验收规范。含后端业务逻辑可测试性、测试分层与集成测试隔离、覆盖率门禁、异步最终一致性的断言方式、缺陷回归用例、任务验收判据要求、归档前验证门禁。
- `delivery/git`: 版本控制与交付约定。含分支模型与变更命名对齐、提交信息规范、提交粒度、PR 描述要素、禁止提交项、换行符策略、版本标记。
- `delivery/ci`: 持续集成门禁约定。含流水线阶段划分（静态检查、单元测试、集成测试、构建、前端检查、规范校验、安全扫描）、门禁判定条件、分支保护要求。

### Modified Capabilities

（无。当前 `openspec/specs/` 为空，不存在既有能力规范被修改）

## Impact

- 新增规范文件：`openspec/specs/backend/layering/spec.md`、`openspec/specs/backend/coding/spec.md`、`openspec/specs/api/contract/spec.md`、`openspec/specs/data/platform/spec.md`、`openspec/specs/web/conventions/spec.md`、`openspec/specs/quality/testing/spec.md`、`openspec/specs/delivery/git/spec.md`、`openspec/specs/delivery/ci/spec.md`
- 新增项目上下文：`openspec/project.md`
- 修改 `openspec/config.yaml` 的 `context` 与 `rules` 两节
- 不涉及任何运行时代码改动，本变更仅产出规范与上下文
- 对后续变更的约束：任何功能变更的 spec 与 design 必须与本基线保持一致；确实需要偏离时，必须以 delta 显式修改对应基线能力并在 proposal 中说明原因，不得在实现中静默绕开
