# Tasks

## 1. 项目上下文与配置落盘

- [x] 1.1 新增 `openspec/project.md`，写入仓库定位四节：是什么、给谁用、解决什么、Non-goals。验证：文件存在且四节齐备，内容与八项能力规范无冲突
- [x] 1.2 更新 `openspec/config.yaml` 的 `context`，写入精简项目定位、技术栈与三条铁律（禁止直接序列化持久化实体 / 事务内禁止数据库之外的 I/O / 对外标识必须字符串化）。验证：执行 `openspec instructions proposal --change establish-engineering-baseline --json`，返回的 `context` 字段包含上述三条
- [x] 1.3 更新 `openspec/config.yaml` 的 `rules`，为 `proposal` 增加规则：必须列出所触及的基线能力，冲突须以增量显式修改基线。验证：同一条命令返回的 `rules.proposal` 包含该规则

## 2. 规范自检

- [x] 2.1 复核八项能力规范中的每条要求都存在明确的检查者与判定方式（可机器强制，或归入人工评审清单），不存在只依赖记忆的模糊约束。验证：对每条要求追问「由谁检查、如何判定违规」，无法回答的要求须改写或删除
- [x] 2.2 执行 `openspec validate establish-engineering-baseline --strict`。验证：输出为 valid 且无警告

## 3. 收尾

- [x] 3.1 执行 `openspec validate --all --strict` 确认全仓规范通过。验证：输出为 valid
- [x] 3.2 归档本变更，使八项能力进入 `openspec/specs/`。验证：`openspec list --specs` 列出八项能力，且 `openspec list` 不再显示本变更
