-- Description: 初始化默认用户、角色、菜单和API资源数据(MYSQL版)
-- Note: 需要有表结构之后再执行此脚本；执行前备份数据，MySQL需支持JSON字段（5.7+）
DELIMITER // -- 临时修改语句结束符，适配存储过程
SET FOREIGN_KEY_CHECKS = 0; -- 关闭外键检查，允许TRUNCATE关联表
START TRANSACTION; -- 开启事务，保证数据原子性


-- 事务提交+恢复外键检查+还原语句结束符
COMMIT;
SET FOREIGN_KEY_CHECKS = 1;
DELIMITER ;
