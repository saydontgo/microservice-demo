# 数据库与 Redis 测试脚本

## 1. 脚本列表

| 脚本 | 说明 |
| --- | --- |
| `mysql_smoke_test.sql` | MySQL 表结构和基础业务写读冒烟测试 |
| `redis_smoke_test.sh` | Redis 连接、TTL、缓存 Key、幂等 Key 冒烟测试 |
| `run_all.sh` | 统一测试入口 |

## 2. 环境变量

### MySQL

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `MYSQL_HOST` | `127.0.0.1` | MySQL 地址 |
| `MYSQL_PORT` | `3306` | MySQL 端口 |
| `MYSQL_USER` | `demo` | MySQL 用户 |
| `MYSQL_PASSWORD` | 空 | MySQL 密码 |
| `MYSQL_DATABASE` | `microservice_demo` | 数据库名 |

### Redis

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `REDIS_HOST` | `127.0.0.1` | Redis 地址 |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | 空 | Redis 密码 |
| `REDIS_DB` | `0` | Redis DB |

## 3. 执行方式

```bash
bash tests/db/run_all.sh all
```

只执行 MySQL：

```bash
bash tests/db/run_all.sh mysql
```

只执行 Redis：

```bash
bash tests/db/run_all.sh redis
```
