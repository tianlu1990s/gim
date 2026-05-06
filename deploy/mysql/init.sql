-- MySQL 初始化脚本，Docker Compose 启动时自动执行
-- 创建数据库和用户，表结构通过 golang-migrate 管理

CREATE DATABASE IF NOT EXISTS gim DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE gim;

CREATE USER IF NOT EXISTS 'gim'@'%' IDENTIFIED BY 'gim_pass';
GRANT ALL PRIVILEGES ON gim.* TO 'gim'@'%';
FLUSH PRIVILEGES;

SET GLOBAL time_zone = '+8:00';

SELECT 'MySQL initialization completed!' AS Status;
