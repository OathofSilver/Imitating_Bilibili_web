# Spec Delta

## Purpose

定义前端工程的目录结构、组件与状态管理边界、路由与数据获取方式、主题与样式约束，使页面模块可预测地落位，避免状态散落与样式硬编码。

## ADDED Requirements

### Requirement: 目录结构约定

前端源码 SHALL 位于仓库的 `web/` 目录，其 `src/` SHALL 按 `app`（路由声明与全局 provider）、`pages`（路由页面）、`components`（跨业务通用组件）、`features`（业务模块）、`store`（Zustand 状态切片）、`api`（接口封装）、`types`（类型定义）、`theme`（Mantine 主题 token）组织。新增文件 MUST 落入上述目录之一，MUST NOT 在源码根目录堆积业务文件。

#### Scenario: 新增一个业务页面

- **WHEN** 开发者新增一个路由页面及其专属子组件
- **THEN** 页面 MUST 位于 `pages`，其专属业务组件 MUST 位于对应 `features` 目录，仅当组件被多个业务复用时 MAY 放入 `components`

#### Scenario: 新增路由或调整主题

- **WHEN** 开发者新增一个可访问路由或修改全局主题配置
- **THEN** 路由声明 MUST 集中在 `app`，主题 token MUST 定义于 `theme`，MUST NOT 分散在页面或组件内部

### Requirement: 组件实现约定

组件 SHALL 使用函数式组件与 hooks 实现，文件名与组件名 SHALL 使用 PascalCase。业务相关逻辑 MUST 放在 `features` 内的业务组件或自定义 hook 中，通用组件 MUST NOT 内含业务接口调用。

#### Scenario: 新增一个通用组件

- **WHEN** 开发者新增一个可复用组件
- **THEN** 该组件 MUST 只接受 props 驱动渲染，MUST NOT 直接调用业务接口

### Requirement: 状态管理边界

Zustand SHALL 仅用于承载跨页面共享的客户端状态（如登录态、播放器状态、全局 UI 状态）。服务端返回的数据 SHALL 由页面或请求层持有，MUST NOT 全量塞入全局 store。

#### Scenario: 新增一个列表页数据

- **WHEN** 页面需要展示接口返回的列表数据
- **THEN** 该数据 MUST 由页面局部状态或请求层持有，MUST NOT 写入全局 store，除非该数据被多个页面共享

### Requirement: 路由与访问守卫

路由 SHALL 由 React Router v7 集中声明。需要登录态的页面 SHALL 在路由层设置统一守卫；未登录访问 MUST 跳转登录并保留原目标地址。权限校验 MUST 在服务端完成，前端守卫仅作为体验优化。

#### Scenario: 未登录访问受限页面

- **WHEN** 未登录用户访问需要登录的路由
- **THEN** 系统 MUST 跳转登录页，并在登录成功后回到原目标地址

### Requirement: 主题与样式约束

样式 SHALL 使用 Mantine 主题 token（颜色、间距、圆角、层级）表达，MUST NOT 硬编码色值；自定义层级（z-index）SHALL 使用主题定义的层级常量，MUST NOT 使用随意数值。

#### Scenario: 新增一处自定义样式

- **WHEN** 开发者需要调整组件颜色或层级
- **THEN** 该值 MUST 来自 Mantine 主题 token 或层级常量，MUST NOT 直接写入十六进制色值或裸 z-index

### Requirement: 类型与接口封装

TypeScript SHALL 开启严格模式，接口入参出参类型 SHALL 在 `types` 中定义并被 `api` 层复用。MUST NOT 使用 `any` 绕过类型检查；确实无法确定的类型 SHALL 使用 `unknown` 并做收窄处理。

#### Scenario: 对接一个新后端接口

- **WHEN** 前端需要对接新的后端接口
- **THEN** 该接口的请求与响应类型 MUST 在 `types` 中声明，并在 `api` 层封装，页面 MUST NOT 直接散落 fetch 调用
