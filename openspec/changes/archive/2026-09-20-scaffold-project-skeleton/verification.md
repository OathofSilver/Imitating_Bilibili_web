# 骨架与基线规范核对记录

核对对象：`openspec/specs/backend/architecture`、`openspec/specs/frontend/conventions`。
核对时间：骨架落地完成后。结论：**无违反项**。

## backend/architecture

| Requirement | 结论 | 依据 |
| --- | --- | --- |
| 三层职责与单向依赖 | 符合 | 每域 `api` 含 handler/logic/svc/config，handler 只做响应封装，logic 承载规则；骨架无业务 SQL |
| 仓库目录布局与单模块组织 | 符合 | `server/` 单一 go.mod；`app/<domain>/{api,rpc,model}` 五域齐备；`common/`、`deploy/` 存在 |
| 对外路径按业务域划分 | 符合 | 五域均暴露 `/api/v1/<domain>/health`，未引入 BFF 聚合层 |
| 服务按业务域划分 | 符合 | user / video / interaction / comment / search 五域 |
| 配置外部化与环境隔离 | 符合 | 各域 `etc/<domain>-api.yaml` 承载 Host/Port/Domain；health 依赖地址走环境变量 |
| 错误逐层包装且不得吞掉 | 部分适用 | `common/response` 与 `common/errcode` 已提供统一封装；尚无跨层错误路径，待业务代码落地时验证 |
| 跨服务调用必须走 RPC | 不适用 | 骨架无跨域调用；`rpc/` 占位并注明后续由 goctl 生成 |

## frontend/conventions

| Requirement | 结论 | 依据 |
| --- | --- | --- |
| 目录结构约定 | 符合 | `web/src` 下 app/pages/features/components/store/api/types/theme 八个目录齐全 |
| 组件实现约定 | 符合 | 函数式组件 + hooks，PascalCase；通用组件 `AppLayout` 不含业务接口调用 |
| 状态管理边界 | 符合 | `store/useAppStore.ts` 仅存 `sidebarCollapsed` 客户端状态；接口数据由页面持有 |
| 路由与访问守卫 | 基本符合 | `app/router.tsx` 集中声明并含 `*` 兜底；未实现登录守卫（骨架不做鉴权，见 design 决策 2，待认证变更） |
| 主题与样式约束 | 符合 | `theme/mantineTheme.ts` 统一 token；组件仅用 `c="pink"`、`gap`、`radius`，无硬编码色值或裸 z-index |
| 类型与接口封装 | 符合 | `types/api.ts` + `api/http.ts` 封装统一响应解析；`tsc --noEmit`（strict）通过 |

## 待后续变更验证的项

1. 错误逐层包装：首个含失败路径的业务接口落地时复核。
2. 跨服务 RPC 调用：首个跨域依赖（如视频详情取作者信息）落地时复核。
3. 路由登录守卫：认证变更落地时按 `identity/auth` 要求实现，并注意服务端强制校验。
