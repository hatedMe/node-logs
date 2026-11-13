-- Compose 首次创建数据卷时执行一次；应用表结构由 Go 服务启动时自动迁移，
-- 此处只保留实例级设置。
SET TIMEZONE = 'UTC';
