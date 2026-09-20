# interaction/model

互动域（点赞/投币/收藏）的数据访问层，只负责存储结构映射与 SQL，不含业务规则。

骨架阶段占位：业务表随各自功能变更建立，迁移脚本放 `server/deploy/migrations/`。
表约定见规范 `data/platform`：Snowflake 对外 ID、软删除、创建与更新时间字段。
