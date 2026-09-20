# platform/skeleton Specification

## Purpose
定义仓库的顶层目录布局、前端与后端工程的落位方式，以及本地一键启动与健康自检能力，使任何开发者都能在同一套命令下验证骨架是否可用。

## Requirements

### Requirement: 顶层目录布局固定

仓库顶层 SHALL 只保留三个代码相关根目录：`web/`（前端）、`server/`（后端）、`openspec/`（规范）。新增代码 MUST 落入其中之一，MUST NOT 在仓库根目录新增业务代码目录。

#### Scenario: 需要新增一段代码

- **WHEN** 开发者新增源码文件
- **THEN** 该文件 MUST 位于 `web/` 或 `server/` 之下，MUST NOT 新建顶层业务目录

### Requirement: 后端目录结构与单模块

后端 SHALL 使用单一 Go module（模块根位于 `server/`）。业务域服务 SHALL 位于 `app/<domain>/` 且包含 `api`、`rpc`、`model` 三个子目录；跨服务公共能力 SHALL 位于 `common/`；部署与数据库脚本 SHALL 位于 `deploy/`。骨架阶段 SHALL 建立五个域：`user`、`video`、`interaction`、`comment`、`search`。

#### Scenario: 检查骨架是否完整

- **WHEN** 开发者检视后端骨架
- **THEN** `app/` 下 MUST 存在五个域目录，每个域 MUST 含 `api`、`rpc`、`model` 三个子目录，`common/` 与 `deploy/` MUST 存在

### Requirement: 前端目录结构

前端 SHALL 位于 `web/`，其 `src/` SHALL 包含 `app`、`pages`、`features`、`components`、`store`、`api`、`types`、`theme` 八个目录。

#### Scenario: 检查前端骨架是否完整

- **WHEN** 开发者检视前端骨架
- **THEN** `web/src/` 下 MUST 存在上述八个目录，且 `app/` 含路由声明、`theme/` 含 Mantine 主题定义

### Requirement: 健康自检接口

每个业务域的 `api` 层 SHALL 提供 `/api/v1/<domain>/health` 接口，返回统一响应结构且 `code` 为 0，`data` 中 SHALL 包含服务名与依赖连通状态。

#### Scenario: 调用健康检查接口

- **WHEN** 请求 `/api/v1/video/health`
- **THEN** 响应 MUST 符合统一响应结构，`code` MUST 为 0，且 `data` 中 MUST 指明该服务名

#### Scenario: 依赖不可用时调用健康检查

- **WHEN** 健康检查被执行而其依赖中间件不可达
- **THEN** 响应 MUST 仍然返回结构化的连通状态（标明不可用），MUST NOT 返回非结构化错误或崩溃

### Requirement: 本地一键启动

项目 SHALL 提供本地启动方式：一条命令启动全部中间件依赖，一条命令启动前端与后端开发服务。健康检查 SHALL 可通过一条命令批量验证全部域。

#### Scenario: 首次克隆仓库后启动

- **WHEN** 开发者克隆仓库并执行依赖启动命令与开发启动命令
- **THEN** 前端 MUST 可在本地访问，后端 MUST 监听并保持可响应

#### Scenario: 批量验证服务健康

- **WHEN** 开发者执行健康自检命令
- **THEN** 五个域的健康状态 MUST 被逐个输出，任一域不可用时 MUST 明确标出该域名
