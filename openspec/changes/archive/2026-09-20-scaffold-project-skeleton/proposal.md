# Proposal

## Why

目录骨架的四项决策已经拍定（`web/` + `server/` 顶层命名、后端单一 Go module、各服务按域前缀直连不引入 BFF、前端 `src/` 补充 `app/` 与 `theme/`），但目前它们只存在于对话与规范文字中，仓库里依然只有 LICENSE 和 README。

没有真实骨架会带来两个问题：`establish-baseline-specs` 里的分层与目录约束无法被实际核对（规范写完没人验证）；后续每个功能变更都要先争论"文件放哪"。本变更把这些决策落成可运行的空骨架——能启动、能自检健康、目录位置明确——让后续功能变更有确定的落点。

## What Changes

- 新增 `web/`：Vite + React 18 + TypeScript 前端工程，包含 `src/{app,pages,features,components,store,api,types,theme}` 八个子目录，接入 Mantine 主题 provider 与 React Router v7 路由声明。
- 新增 `server/`：单一 Go module，`common/`（响应封装、错误码、ID 生成、中间件）+ `app/{user,video,interaction,comment,search}/{api,rpc,model}` + `deploy/`。
- 为每个业务域的 `api` 层提供 `/api/v1/<domain>/health` 健康检查，返回统一响应结构，用于验证骨架联通。
- 新增 `deploy/docker-compose.yaml`（MySQL、Redis、RabbitMQ、Elasticsearch）与 `Makefile`（dev / up / health）。
- 更新 README，说明目录结构与本地启动方式。

本变更**不**实现任何业务功能（推荐流、播放器、评论等），只交付可运行的空壳。

## Capabilities

### New Capabilities

- `platform/skeleton`：仓库目录布局约定、本地启动方式与健康自检能力

### Modified Capabilities

（无）

## Impact

- 新增目录与文件：`web/`、`server/`、`deploy/docker-compose.yaml`、`Makefile`。
- 依赖：Node 22（pnpm）、Go 1.22+、Docker（仅本地中间件）；goctl 若可用则用于生成 go-zero 代码骨架，不可用时以等价手工结构替代。
- 影响后续变更：所有功能变更的目录落位 MUST 与本骨架一致；新增业务域 MUST 复用 `app/<domain>/{api,rpc,model}` 结构。
- 不影响任何线上系统（项目尚无部署环境）。
