# MySQL 建表与测试步骤

## 1. 目标

本文档说明如何在目标运行环境中创建电商微服务 Demo 所需的 MySQL 表，并在建表完成后执行 MySQL 与 Redis 测试脚本。

当前本机不是运行环境。目标运行环境仍按 PRD 约定为 Windows 11 + WSL2。

## 2. 文件说明

| 文件 | 说明 |
| --- | --- |
| `scripts/database/create_tables.sql` | 创建全部 MySQL 表、索引、约束 |
| `scripts/database/drop_tables.sql` | 删除全部 Demo 表，调试时谨慎使用 |
| `tests/db/mysql_smoke_test.sql` | MySQL 建表后冒烟测试脚本 |
| `tests/db/redis_smoke_test.sh` | Redis 连通性、Key、TTL、幂等缓存冒烟测试 |
| `tests/db/run_all.sh` | 串行执行 MySQL 与 Redis 测试 |

## 3. 前置条件

目标 WSL2 环境需要具备：

- MySQL 8.0+。
- Redis 6.0+。
- `mysql` 命令行客户端。
- `redis-cli` 命令行客户端。
- Bash。

## 4. 创建数据库

示例命令：

```bash
mysql -h 127.0.0.1 -P 3306 -uroot -p -e "CREATE DATABASE IF NOT EXISTS microservice_demo DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

如使用普通用户，建议提前授权：

```sql
CREATE USER IF NOT EXISTS 'demo'@'%' IDENTIFIED BY 'demo_password';
GRANT ALL PRIVILEGES ON microservice_demo.* TO 'demo'@'%';
FLUSH PRIVILEGES;
```

## 5. 执行建表

```bash
mysql -h 127.0.0.1 -P 3306 -udemo -p microservice_demo < scripts/database/create_tables.sql
```

建表脚本会创建以下表：

- `users`
- `buyer_profiles`
- `seller_profiles`
- `products`
- `product_inventory`
- `orders`
- `payment_transactions`
- `product_daily_stats`
- `seller_daily_stats`
- `idempotency_records`

## 6. 表结构检查

```bash
mysql -h 127.0.0.1 -P 3306 -udemo -p microservice_demo -e "SHOW TABLES;"
mysql -h 127.0.0.1 -P 3306 -udemo -p microservice_demo -e "SHOW CREATE TABLE orders\G"
```

重点检查：

- `status`、`type` 类字段均为 `INT`。
- 商品、订单、支付流水、幂等记录具备 `CHECK` 约束。
- 订单表存在 `idx_orders_buyer_status_created`、`idx_orders_seller_product_status`、`idx_orders_status_created` 索引。

## 7. 执行 MySQL 测试

```bash
MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=demo \
MYSQL_PASSWORD=demo_password \
MYSQL_DATABASE=microservice_demo \
bash tests/db/run_all.sh mysql
```

MySQL 测试会验证：

- 10 张表均已创建。
- `status`、`type` 字段均为 `INT`。
- 买家、卖家、商品、库存、订单、支付流水、统计、幂等记录可正常写入。
- 商品状态、订单状态、支付类型均使用数字枚举。
- 订单默认查询 `status IN (1, 2)` 可返回未完成订单。

## 8. 执行 Redis 测试

```bash
REDIS_HOST=127.0.0.1 \
REDIS_PORT=6379 \
REDIS_DB=0 \
bash tests/db/run_all.sh redis
```

如果 Redis 设置了密码：

```bash
REDIS_PASSWORD=your_password bash tests/db/run_all.sh redis
```

Redis 测试会验证：

- Redis 连接可用。
- 登录 Token Key 可写入并设置 TTL。
- 商品搜索缓存 Key 使用数字状态编码。
- 幂等 Key 可通过 `SET NX EX` 防止重复写入。
- 测试 Key 会在执行结束后清理。

## 9. 执行全部测试

```bash
MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=demo \
MYSQL_PASSWORD=demo_password \
MYSQL_DATABASE=microservice_demo \
REDIS_HOST=127.0.0.1 \
REDIS_PORT=6379 \
REDIS_DB=0 \
bash tests/db/run_all.sh all
```

## 10. 回滚建表

如需要重新初始化表，可执行：

```bash
mysql -h 127.0.0.1 -P 3306 -udemo -p microservice_demo < scripts/database/drop_tables.sql
mysql -h 127.0.0.1 -P 3306 -udemo -p microservice_demo < scripts/database/create_tables.sql
```

注意：`drop_tables.sql` 会删除所有 Demo 表和数据，仅用于本地开发或测试环境。

## 11. 常见问题

### 11.1 MySQL 报 CHECK 语法或约束问题

确认 MySQL 版本为 8.0+。如果使用较旧版本，`CHECK` 可能不会被执行或语义不一致。

### 11.2 Redis 测试报 NOAUTH

说明 Redis 开启了密码认证，请设置：

```bash
export REDIS_PASSWORD=your_password
```

### 11.3 MySQL 测试失败后有测试数据残留

`mysql_smoke_test.sql` 默认在事务内执行并回滚。若连接中断，可按测试用户名或商品名前缀 `smoke_` 手动清理。
