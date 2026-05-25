-- 电商微服务架构 Demo MySQL 删表脚本
-- 谨慎执行: 会删除所有 Demo 表。

SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS seller_daily_stats;
DROP TABLE IF EXISTS product_daily_stats;
DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS product_inventory;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS seller_profiles;
DROP TABLE IF EXISTS buyer_profiles;
DROP TABLE IF EXISTS users;
SET FOREIGN_KEY_CHECKS = 1;
