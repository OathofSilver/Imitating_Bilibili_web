# Tasks

## 1. 前端骨架

- [x] 1.1 在 `web/` 初始化 Vite + React 18 + TypeScript 工程并安装 Mantine、Zustand、React Router v7（验收：`pnpm install` 成功，`pnpm dev` 能启动开发服务器并返回 200）
- [x] 1.2 建立 `src/{app,pages,features,components,store,api,types,theme}` 八个目录并各放入口文件（验收：八个目录均存在且非空）
- [x] 1.3 在 `theme/` 定义 Mantine 主题 token，在 `app/` 声明路由与 provider 并渲染首页（验收：首页能渲染 Mantine 组件，访问未知路径命中路由兜底页）
- [x] 1.4 配置 TypeScript strict 与路径别名（验收：`pnpm tsc --noEmit` 通过且无 `any`）

## 2. 后端骨架

- [x] 2.1 初始化 `server/go.mod` 并建 `common/`、`deploy/` 目录（验收：`go build ./...` 成功，模块名与目录名一致）
- [x] 2.2 在 `common/` 实现统一响应封装、错误码表、Snowflake ID 生成与请求中间件占位（验收：`go vet ./common/...` 通过，响应结构字段为 code/message/data）
- [x] 2.3 建立 `app/{user,video,interaction,comment,search}/{api,rpc,model}` 五个域三层目录，`rpc` 与 `model` 放占位说明文件（验收：五个域目录齐全，每域三个子目录存在）
- [x] 2.4 为五个域的 `api` 层各实现 `/api/v1/<domain>/health` 并返回统一响应（验收：逐个 curl 五个路径，均返回 `code` 为 0 且 data 含服务名）
- [x] 2.5 `health` 在依赖不可达时仍返回结构化状态（验收：停掉 Redis 后调用 health，响应仍是统一结构且标明依赖不可用，服务不崩溃）

## 3. 本地环境与文档

- [x] 3.1 编写 deploy/docker-compose.yaml 编排 MySQL、Redis、RabbitMQ、Elasticsearch（验收：docker compose up 后四个服务端口可连通，五域 health dependencies 全为 true）
- [x] 3.2 编写 `Makefile`，提供 `up`、`dev`、`health` 三个目标（验收：`make health` 输出五个域的健康状态）
- [x] 3.3 更新 README，写入目录结构说明与本地启动步骤（验收：README 含目录树与启动命令，按文档可复现启动）

## 4. 骨架与规范核对

- [x] 4.1 逐条核对 `backend/architecture` 与 `frontend/conventions` 的目录与分层约束（验收：核对结论记录在本变更中，列出全部违反项或明确"无违反"）
- [x] 4.2 执行 `openspec validate scaffold-project-skeleton --strict` 通过（验收：命令退出码为 0）
