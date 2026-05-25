# 数据模型总览

## 1. 设计目标

数据库以 MySQL 为主存储，Redis 做缓存和幂等控制。表设计围绕用户、商品、库存、订单、支付、统计六个领域展开。

## 2. 核心实体

| 实体 | 表 | 说明 |
| --- | --- | --- |
| 用户账号 | `users` | 登录账号、密码摘要、角色 |
| 买家资料 | `buyer_profiles` | 昵称、头像、手机号、地址、余额展示关联 |
| 卖家资料 | `seller_profiles` | 注册人、店铺名称、头像、主题、成交总额 |
| 商品 | `products` | 商品基础信息、价格、状态、卖家归属 |
| 库存 | `product_inventory` | 可售库存、预占库存、累计发货量 |
| 订单 | `orders` | 买家、卖家、商品、数量、金额、状态 |
| 支付流水 | `payment_transactions` | 充值、支付、退款流水 |
| 商品日统计 | `product_daily_stats` | 商品维度成交、退款和订单量 |
| 卖家日统计 | `seller_daily_stats` | 卖家维度趋势图数据 |
| 幂等记录 | `idempotency_records` | 写操作幂等控制 |

## 3. 关系说明

```text
users 1--1 buyer_profiles
users 1--1 seller_profiles
seller_profiles 1--N products
products 1--1 product_inventory
buyer_profiles 1--N orders
seller_profiles 1--N orders
products 1--N orders
orders 1--N payment_transactions
products 1--N product_daily_stats
seller_profiles 1--N seller_daily_stats
```

## 4. 金额字段规范

所有金额字段使用分作为单位，类型使用 `BIGINT`，避免浮点误差。

| 字段语义 | 示例 |
| --- | --- |
| 商品单价 10.99 元 | `price_cent = 1099` |
| 订单金额 100 元 | `total_amount_cent = 10000` |
| 1 亿上限 | `10000000000` 分 |

## 5. 时间字段规范

所有核心表包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `created_at` | `DATETIME(3)` | 创建时间 |
| `updated_at` | `DATETIME(3)` | 更新时间 |
| `biz_date` | `DATE` | 统计或分区业务日期，按需使用 |

## 6. 删除策略

Demo 不做物理删除。商品下架、订单退款等均通过状态字段表达。

## 7. 枚举字段规范

数据库中所有 `status`、`type` 类字段统一使用 `INT` 编码，具体映射见 `mysql-tables.md`。服务层和前端可以展示字符串标签，但 MySQL 不存储字符串枚举。
