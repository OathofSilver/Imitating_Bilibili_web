# Tasks

## 1. 评审基线规范内容

- [x] 1.1 评审 `backend/architecture`：确认三层划分、五个业务域、RPC 调用约束符合预期（验收：逐条确认无异议，或与用户修订后写回文件）
- [x] 1.2 评审 `api/contract`：确认统一响应结构、错误码区段（0/4xxxx/5xxxx）、分页字段、幂等要求（验收：`code/message/data` 与错误码分段被明确确认）
- [x] 1.3 评审 `data/platform`：确认 Snowflake ID、软删除、Redis 计数异步落库、MQ 幂等、ES 降级策略（验收：五项策略均被确认，无遗漏）
- [x] 1.4 评审 `frontend/conventions`：确认目录结构、Zustand 边界、路由守卫、Mantine token 约束（验收：目录命名与状态边界被明确确认）
- [x] 1.5 评审 `identity/auth`：确认凭证签发/过期处理、服务端强制鉴权、等级权限模型、脱敏要求（验收：四项要求均被确认）
- [x] 1.6 评审 `quality/testing`：确认测试范围与"每个任务必须带验收步骤"的要求（验收：验收判据要求被确认）

## 2. 校验与归档

- [x] 2.1 执行 `openspec validate establish-baseline-specs --strict` 并通过（验收：命令退出码为 0，无格式或完整性错误）
- [x] 2.2 执行 `openspec archive establish-baseline-specs --yes` 完成归档（验收：`openspec/specs/` 下出现 backend/architecture、api/contract、data/platform、frontend/conventions、identity/auth、quality/testing 六个目录且各含 spec.md）
- [x] 2.3 执行 `openspec list --specs` 确认六个能力已登记为当前规范（验收：输出列出全部六个 capability）

## 3. 让基线规范生效

- [x] 3.1 更新 `openspec/project.md`，声明工程约束以 `openspec/specs/` 为准，冲突时以 specs 为准（验收：project.md 中出现该声明且与 specs 无矛盾表述）
- [x] 3.2 在 `openspec/config.yaml` 的 `rules.proposal` 中补充"新变更必须列出所触及的基线规范"（验收：config.yaml 中该规则存在，`openspec instructions proposal` 输出包含此规则）
- [x] 3.3 创建首个工程骨架变更时，在其 design 中逐条核对 `backend/architecture` 与 `frontend/conventions`（验收：骨架变更的 design.md 包含核对清单，且骨架目录结构与规范一致）
