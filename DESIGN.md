# 电商微服务架构 Demo 总体设计

## 1. 背景与目标

本项目是一个单机运行的电商微服务架构 Demo，后端语言为 Go，前端使用 HTML、CSS、JavaScript 组成的 Web 应用。系统面向两类用户：买家与卖家。用户在登录页选择身份后进入对应工作台。

目标不是建设生产级电商平台，而是在单机环境中完整演示：

- 买家注册、登录、浏览商品、下单支付、订单查看、确认收货、退款、充值。
- 卖家注册、登录、店铺资料维护、商品管理、下架、补库存、一键发货、经营趋势查看。
- MySQL 存储核心交易数据，Redis 承担缓存、会话和幂等控制。
- 通过清晰服务边界模拟微服务架构，同时保持单机可运行复杂度。

## 2. 目标运行环境

本机不是运行环境。目标运行环境按 PRD 设定为：

| 项目 | 说明 |
| --- | --- |
| OS | Windows 11 + WSL2 |
| CPU | AMD 7800X3D，8 核 16 线程 |
| 内存 | 48GB |
| 部署形态 | 单机，多进程或 Docker Compose 均可 |
| 数据库 | MySQL |
| 缓存 | Redis |

## 3. 总体架构

系统采用“前端静态应用 + API 网关 + 后端领域服务 + MySQL + Redis”的结构。

```text
Browser
  |
  v
Frontend Static App
  |
  v
API Gateway / BFF
  |
  +--> Auth Service
  +--> User Service
  +--> Product Service
  +--> Order Service
  +--> Payment Service
  +--> Inventory Service
  +--> Analytics Service
  |
  +--> Redis
  +--> MySQL
```

## 4. 服务划分

| 服务 | 职责 |
| --- | --- |
| Auth Service | 注册、登录、身份鉴权、Token 签发、退出登录 |
| User Service | 买家资料、卖家资料、店铺资料、主题设置 |
| Product Service | 商品创建、查询、状态流转、前缀搜索、卖家商品列表 |
| Inventory Service | 库存查询、补库存、下单扣减、发货库存校验 |
| Order Service | 下单、订单查询、发货、确认收货、退款、订单状态流转 |
| Payment Service | 买家余额、充值、支付扣款、退款入账、卖家成交额修正 |
| Analytics Service | 卖家成交金额、退款金额、退款率、趋势图统计 |

## 5. 核心业务流程

### 5.1 登录注册

1. 用户选择身份：买家或卖家。
2. 新用户注册时写入 `users` 表，并根据身份写入 `buyer_profiles` 或 `seller_profiles`。
3. 登录成功后生成访问 Token，Token 中包含 `user_id` 与 `role`。
4. 服务端鉴权中间件校验 Token 与接口角色要求。

### 5.2 买家下单支付

1. 买家通过商品名前缀检索商品，仅返回 `1`（在售） 和 `2`（预售中） 状态商品。
2. 买家选择购买数量，系统校验数量不超过 100，总金额不超过 1 亿。
3. 系统校验余额足够后扣减买家余额。
4. 对非预售商品执行库存预扣；预售商品库存展示为 0，但可允许创建待发货订单。
5. 创建订单，状态为 `1`（已下单未发货）。
6. 更新商品统计与卖家成交总额。

### 5.3 买家退款

1. 仅 `1`（已下单未发货） 订单允许退款。
2. 使用幂等键防止重复退款。
3. 买家余额增加订单金额。
4. 卖家成交总额减少。
5. 订单状态更新为 `4`（已退款），并记录退款金额。

### 5.4 卖家一键发货

1. 卖家在商品管理列表选择商品。
2. 查询该商品所有 `1`（已下单未发货） 订单总数量。
3. 检查库存是否大于等于待发货数量。
4. 库存不足时提示先补库存。
5. 库存充足时批量更新订单为 `2`（配送中），扣减可售库存。

### 5.5 商品下架

1. 仅在售商品可点击下架。
2. 下架后商品状态变为 `3`（正在下架），买家侧不可见。
3. 卖家侧仍可补库存并发货。
4. 当该商品不存在未完成订单时，状态可转为 `4`（已下架）。

## 6. 数据分区原则

PRD 要求所有表默认以时间分区，除特殊说明外均使用 `created_at` 或业务时间作为分区键。订单列表有特殊要求，按订单状态分区，因此订单表采用 `order_status + created_at` 的组合分区设计。

数据库表中的 `status`、`type` 类字段统一使用 `INT` 枚举编码，不存储字符串枚举。具体编码见 `docs/01-database/mysql-tables.md` 的枚举编码规范。

## 7. 鉴权与幂等

- 鉴权：所有业务接口都要求 Bearer Token。
- 角色隔离：买家接口只允许 `BUYER`，卖家接口只允许 `SELLER`。
- 数据隔离：卖家只能访问自己的店铺和商品，买家只能访问自己的订单与余额。
- 幂等：注册、支付、退款、一键发货、充值等写操作要求 `Idempotency-Key`。

## 8. 文档索引

| 文档 | 说明 |
| --- | --- |
| `docs/00-overview/product-scope.md` | 产品范围和非目标 |
| `docs/00-overview/architecture.md` | 架构设计 |
| `docs/00-overview/module-boundary.md` | 模块边界 |
| `docs/01-database/schema-design.md` | 数据模型总览 |
| `docs/01-database/mysql-tables.md` | MySQL 表字段 |
| `docs/01-database/redis-cache.md` | Redis Key 设计 |
| `docs/01-database/partition-strategy.md` | 分区策略 |
| `docs/01-database/query-sql.md` | 核心取数 SQL |
| `docs/02-api/*.md` | API 设计 |
| `docs/03-services/*.md` | 服务设计 |
| `docs/04-frontend/*.md` | 前端设计 |
| `docs/05-runtime/*.md` | 运行与配置说明 |
| `docs/06-quality/*.md` | 质量保障设计 |
