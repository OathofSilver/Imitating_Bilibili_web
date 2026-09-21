-- 用户账号与资料表。
-- 幂等：可重复执行。
-- 执行方式（bilibili_web 库已由 deploy/docker-compose.yaml 创建）：
--   mysql -h 127.0.0.1 -P 13306 -u root -proot bilibili_web < server/deploy/users.sql
-- 本文件同时是 goctl 生成 model 层的输入：
--   goctl model mysql ddl -src deploy/users.sql -dir app/user/model

CREATE TABLE IF NOT EXISTS `users` (
  `id`            BIGINT       NOT NULL COMMENT 'Snowflake 全局 ID，唯一对外标识，非自增',
  `username`      VARCHAR(32)  NOT NULL COMMENT '登录用户名，大小写不敏感唯一',
  `password_hash` VARCHAR(72)  NOT NULL COMMENT 'bcrypt 哈希值，任何响应都不得返回',
  `nickname`      VARCHAR(32)  NOT NULL DEFAULT '' COMMENT '昵称，注册时默认取用户名',
  `avatar_url`    VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像地址，仅存 URL 字符串',
  `signature`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '个性签名',
  `gender`        TINYINT      NOT NULL DEFAULT 0 COMMENT '性别 0 未知 1 男 2 女',
  `birthday`      DATE         NULL COMMENT '生日',
  `level`         INT          NOT NULL DEFAULT 0 COMMENT '用户等级',
  `role`          TINYINT      NOT NULL DEFAULT 0 COMMENT '角色 0 普通用户 1 管理员',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at`    DATETIME     NULL DEFAULT NULL COMMENT '软删除标记，非空表示已删除，业务数据禁止物理删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户账号与资料';
