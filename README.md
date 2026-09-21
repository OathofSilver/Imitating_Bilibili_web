# Imitating-Bilibili-web

模仿 B 站 Web 端项目：高保真复刻核心页面与交互，前后端一体化实现（非静态 mock）。

本项目采用规范驱动开发（OpenSpec），`openspec/specs/` 是系统行为的唯一事实来源，
所有功能先提变更提案，人工确认后再实施。

## 技术栈

- 前端：React 18 + TypeScript + Vite 7 + Mantine + Zustand + React Router v7（pnpm）
- 后端：Go + go-zero + MySQL + Redis + RabbitMQ + Elasticsearch（Snowflake 生成全局 ID）

## 目录结构

```
Imitating_Bilibili_web/
├── web/                      前端
│   └── src/{app,pages,features,components,store,api,types,theme}
├── server/                   后端（单一 Go module）
│   ├── common/               响应封装 · 错误码 · Snowflake · 中间件 · 健康检查
│   ├── app/<domain>/{api,rpc,model}/   五个业务域，各自三层
│   └── deploy/               docker-compose · 数据库迁移
├── openspec/                 规范（specs/ 为事实来源，changes/ 为进行中的变更）
└── Makefile                  up / dev / health 等常用命令
```

业务域与端口：

| 域 | 端口 | 健康检查 |
| --- | --- | --- |
| user | 8001 | `/api/v1/user/health` |
| video | 8002 | `/api/v1/video/health` |
| interaction | 8003 | `/api/v1/interaction/health` |
| comment | 8004 | `/api/v1/comment/health` |
| search | 8005 | `/api/v1/search/health` |

## 本地启动

```bash
make up        # 启动 MySQL / Redis / RabbitMQ / Elasticsearch
make dev       # 启动前端(5173) 与五个后端域服务(8001-8005)
make health    # 逐个调用五个域的健康检查接口
make down      # 停止中间件容器
```

不使用 make 时：

```bash
docker compose -f server/deploy/docker-compose.yaml up -d   # 或 docker-compose（本机为独立版）
pnpm -C web dev
cd server/app/video/api && go run .                          # 每个域单独启动
```

中间件端口：MySQL `13306`、Redis `6379`、RabbitMQ `5673`（管理台 `15673`）、Elasticsearch `9200`。
MySQL 与 RabbitMQ 使用备用端口是因为宿主机 3306/5672 已被本机服务占用，且 3307 段落在 Windows 保留端口内；
需要调整时同时改 `server/deploy/docker-compose.yaml` 与 `Makefile` 顶部的地址变量。

健康检查在依赖未就绪时仍返回 `code: 0`，仅在 `data.dependencies` 中标注各中间件连通状态。

## 工程约定

- API 统一前缀 `/api/v1/<domain>/<resource>`，响应结构 `{ code, message, data }`
- 后端分层：api（参数校验）→ rpc（业务规则）→ model（数据访问），依赖单向
- 对外 ID 一律由 Snowflake 生成，不使用数据库自增主键
- 详细约束见 `openspec/specs/`（backend/architecture、api/contract、data/platform、
  frontend/conventions、identity/auth、quality/testing）
