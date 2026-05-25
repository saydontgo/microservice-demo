# 配置设计

## 1. 配置来源

推荐使用环境变量或 `.env` 文件。当前阶段只定义配置项，不创建运行代码。

## 2. 数据库配置

| 配置 | 示例 | 说明 |
| --- | --- | --- |
| `MYSQL_HOST` | `127.0.0.1` | MySQL 地址 |
| `MYSQL_PORT` | `3306` | MySQL 端口 |
| `MYSQL_DATABASE` | `microservice_demo` | 数据库名 |
| `MYSQL_USER` | `demo` | 用户名 |
| `MYSQL_PASSWORD` | `demo_password` | 密码 |

## 3. Redis 配置

| 配置 | 示例 | 说明 |
| --- | --- | --- |
| `REDIS_HOST` | `127.0.0.1` | Redis 地址 |
| `REDIS_PORT` | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | 空 | 密码 |
| `REDIS_DB` | `0` | DB 编号 |

## 4. 服务配置

| 配置 | 示例 | 说明 |
| --- | --- | --- |
| `HTTP_PORT` | `8000` | 后端入口端口 |
| `FRONTEND_PORT` | `8080` | 前端静态服务端口 |
| `TOKEN_TTL_SECONDS` | `7200` | Token 有效期 |
| `IDEMPOTENCY_TTL_SECONDS` | `86400` | 幂等记录有效期 |
| `MAX_BUY_QUANTITY` | `100` | 单次购买数量上限 |
| `MAX_ORDER_AMOUNT_CENT` | `10000000000` | 单次订单金额上限，分 |

## 5. 安全配置

| 配置 | 说明 |
| --- | --- |
| `PASSWORD_SALT` | 密码摘要盐值 |
| `TOKEN_SECRET` | Token 签名密钥 |
| `CORS_ALLOWED_ORIGINS` | 前端允许跨域来源 |
