# Proposal

## Why

当前仓库是空项目，技术栈（React 18 + TypeScript + Vite 7 + Mantine + Zustand + React Router v7 / Go + go-zero + MySQL + Redis + RabbitMQ + Elasticsearch + Snowflake）与项目定位（高保真复刻 B 站 Web 端，前后端一体化）已经确定，但只存在于 `openspec/project.md` 这类提示词级上下文中。它不参与 `openspec validate`，也不会被 delta 追踪，因此对后续 AI 生成的代码没有任何硬约束力。

结果是：第一个功能变更开始时，分层方式、ID 生成、缓存与异步边界、API 响应格式这些关键决策都要重新拍一次，且每次可能拍出不同结果。本变更把这些决策一次性固化为可执行、可校验的基线规范，让后续所有功能变更（首页推荐流、播放器、评论、搜索等）都在同一套约束下展开。

现在做是因为代码量还为零——此时固化约束的成本最低，一旦骨架落地再回头统一，就要付出重构代价。

## What Changes

建立六项基线能力规范（均为新增，无既有规范被修改）：

- `backend/architecture`：go-zero 服务分层（api → rpc → model）、依赖方向、跨服务调用约束与禁止事项。
- `api/contract`：统一响应结构、版本前缀、错误码体系、分页与幂等约定。
- `data/platform`：Snowflake 全局 ID、MySQL 表与事务约定、Redis 缓存与计数策略、RabbitMQ 异步边界、Elasticsearch 检索与最终一致性。
- `frontend/conventions`：目录结构、组件与状态管理约定、路由与数据获取、Mantine 主题与样式约束。
- `identity/auth`：登录态载体与鉴权流程、用户等级与权限模型的基础约定。
- `quality/testing`：后端与前端测试要求，以及每个任务必须携带可验证验收步骤的规范。

这些规范描述的是**工程约束**，其验收方式是代码结构检查、评审与自动化校验，而非运行时功能测试。

## Capabilities

### New Capabilities

- `backend/architecture`：后端服务分层与依赖方向约束（api / rpc / model 三层职责与禁止事项）
- `api/contract`：前后端 HTTP 接口契约（统一响应、版本前缀、错误码、分页、幂等）
- `data/platform`：数据层与中间件使用策略（Snowflake ID、MySQL、Redis、RabbitMQ、Elasticsearch）
- `frontend/conventions`：前端工程约定（目录结构、组件、状态管理、路由、主题与样式）
- `identity/auth`：认证与用户体系基础约定（登录态、鉴权、用户等级与权限模型）
- `quality/testing`：测试与验收规范（测试覆盖要求、验收标准、变更完成判定）

### Modified Capabilities

（无，当前 `openspec/specs/` 为空，不存在既有能力规范被修改）

## Impact

- 新增规范文件：`openspec/specs/{backend/architecture,api/contract,data/platform,frontend/conventions,identity/auth,quality/testing}/spec.md`
- 影响后续所有变更：任何新功能变更的 spec 与 design 都必须与上述基线规范保持一致；冲突时须显式声明并以 delta 修改基线规范，不得静默违反。
- 影响代码骨架落地：首个工程骨架变更需按 `backend/architecture` 与 `frontend/conventions` 组织目录与模块。
- 不涉及运行时代码改动，本变更仅产出规范。
