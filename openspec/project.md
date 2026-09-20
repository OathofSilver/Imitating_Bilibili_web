# Project Context

## Purpose

高保真复刻哔哩哔哩（bilibili）Web 端的核心体验，前后端一体化实现：
前端还原首页推荐流、视频播放页、搜索、评论区、个人空间等主要页面与交互；
后端提供对应的真实 API，而非静态 mock。

目标不是像素级抄袭，而是用工程化的方式复现 B 站的核心产品形态与技术方案，
用于完整演练「前端交互 + 高并发后端」的链路设计。

## Tech Stack

前端：
- React 18 + TypeScript + Vite 7
- UI 组件库：Mantine
- 状态管理：Zustand
- 路由：React Router v7
- 包管理：pnpm

后端：
- Go + go-zero 框架
- 存储：MySQL（主数据）
- 缓存：Redis（热点数据、计数、会话）
- 消息：RabbitMQ（异步管线：转码回调、通知、计数落库）
- 检索：Elasticsearch（视频/用户搜索）
- 全局 ID：Snowflake

## Architecture Conventions

- 前后端分离：前端通过 REST API 访问后端，统一前缀 `/api/v1`。
- 后端按 go-zero 分层：`api`（HTTP 网关与参数校验）→ `rpc`（领域服务）→ `model`（数据访问）。
  跨服务调用走 RPC，禁止 api 层直接写业务 SQL。
- 前端目录：`src/pages`（页面）、`src/components`（通用组件）、`src/features`（业务模块）、
  `src/store`（Zustand slices）、`src/api`（接口封装）、`src/types`。
- 所有全局 ID 由 Snowflake 生成，禁止使用数据库自增主键对外暴露。
- 写多读少的计数类数据（播放量、点赞、弹幕数）走 Redis 累加 + RabbitMQ 异步落库，
  不直接同步写 MySQL。
- 搜索相关读请求走 Elasticsearch，MySQL 为数据源，通过异步任务保证最终一致。

## Code Style

- 前端：TypeScript strict 模式，组件使用函数式 + hooks，导出组件用 PascalCase 文件名；
  样式优先使用 Mantine 主题 token，避免硬编码色值。
- 后端：Go 标准命名与 `gofmt`，error 必须逐层包装携带上下文，禁止吞掉错误；
  go-zero 的 `logic` 层承载业务，handler 只做参数绑定与响应封装。
- 统一响应结构：`{ code, message, data }`，HTTP 状态码与业务 code 分离。

## Testing Strategy

- 后端：`go test` + table-driven 用例，`logic` 层依赖通过接口注入以便 mock。
- 前端：关键交互与状态流转（feed 加载、播放器状态机、评论区）需有测试覆盖。
- 每个变更在 `tasks.md` 中必须包含可验证的验收步骤，不能只写「完成实现」。

## OpenSpec Workflow

- 工程约束以 `openspec/specs/` 为唯一事实来源；本文件仅作总体概述，与 specs 冲突时 MUST 修改本文件使其与 specs 一致。
- 所有功能先提变更提案（`/opsx:propose`），按 proposal → specs → design → tasks 推进，
  人工确认后再实施、归档。
- 规范文档正文使用中文书写；结构性标题与 RFC 2119 关键字（SHALL / MUST / SHOULD / MAY）保留英文。
- 需求条目必须带 Given/When/Then 验收场景，否则视为不完整。
- `openspec/specs/` 是系统当前行为的唯一事实来源，任何行为变化都通过变更增量（delta）描述。

## Current State

仓库当前为空（仅 LICENSE 与 README）。技术栈与架构约定已确定，尚未落地任何代码。
建议的首个变更：搭建前后端工程骨架（前端 Vite 项目初始化 + 后端 go-zero 服务骨架与数据库基础表）。
