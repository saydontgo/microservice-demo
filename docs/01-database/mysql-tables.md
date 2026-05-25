# MySQL 表字段设计

## 1. 枚举编码规范

数据库表中的 `status`、`type` 类字段统一使用 `INT` 编码，不使用字符串枚举。服务层可将编码映射为业务常量，API 响应可同时返回编码和展示文案。

### 1.1 用户状态 `users.status`

| 编码 | 常量 | 说明 |
| --- | --- | --- |
| `1` | `ACTIVE` | 正常 |
| `2` | `DISABLED` | 禁用 |

### 1.2 商品状态 `products.status`

| 编码 | 常量 | 说明 | 买家可见 |
| --- | --- | --- | --- |
| `1` | `ON_SALE` | 在售 | 是 |
| `2` | `PRE_SALE` | 预售中 | 是 |
| `3` | `DELISTING` | 正在下架 | 否 |
| `4` | `OFF_SHELF` | 已下架 | 否 |

### 1.3 订单状态 `orders.status`

| 编码 | 常量 | 说明 |
| --- | --- | --- |
| `1` | `PLACED_UNSHIPPED` | 已下单未发货 |
| `2` | `SHIPPING` | 配送中 |
| `3` | `RECEIVED` | 已收货 |
| `4` | `REFUNDED` | 已退款 |

### 1.4 支付流水类型 `payment_transactions.type`

| 编码 | 常量 | 说明 |
| --- | --- | --- |
| `1` | `RECHARGE` | 充值 |
| `2` | `PAYMENT` | 支付扣款 |
| `3` | `REFUND` | 退款入账 |

### 1.5 处理状态 `payment_transactions.status` / `idempotency_records.status`

| 编码 | 常量 | 说明 |
| --- | --- | --- |
| `1` | `PROCESSING` | 处理中，仅幂等记录使用 |
| `2` | `SUCCESS` | 成功 |
| `3` | `FAILED` | 失败 |

## 2. `users`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT` | PK | 用户 ID |
| `username` | `VARCHAR(64)` | UNIQUE | 登录名 |
| `password_hash` | `VARCHAR(255)` | NOT NULL | 密码摘要 |
| `role` | `VARCHAR(16)` | NOT NULL | `BUYER` 或 `SELLER`，角色不属于 status/type 字段，保留字符串便于鉴权可读性 |
| `status` | `INT` | NOT NULL DEFAULT 1 | 用户状态：1 正常，2 禁用 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 3. `buyer_profiles`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `user_id` | `BIGINT` | PK | 关联 `users.id` |
| `nickname` | `VARCHAR(64)` | NOT NULL | 买家昵称 |
| `avatar_url` | `VARCHAR(512)` | NULL | 买家头像 |
| `phone` | `VARCHAR(32)` | NULL | 手机号 |
| `shipping_address` | `VARCHAR(512)` | NULL | 收货地址 |
| `balance_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 当前余额，单位分 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 4. `seller_profiles`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `user_id` | `BIGINT` | PK | 关联 `users.id` |
| `registrant_name` | `VARCHAR(64)` | NOT NULL | 注册人 |
| `shop_name` | `VARCHAR(128)` | NOT NULL | 店铺名称 |
| `shop_avatar_url` | `VARCHAR(512)` | NULL | 店铺头像 |
| `theme` | `VARCHAR(16)` | NOT NULL DEFAULT `LIGHT` | 页面主题，不属于 status/type 字段 |
| `total_deal_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 成交商品总额 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 5. `products`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT` | PK | 商品 ID |
| `seller_id` | `BIGINT` | INDEX | 卖家用户 ID |
| `product_name` | `VARCHAR(128)` | INDEX | 商品名，支持前缀匹配 |
| `description` | `VARCHAR(1024)` | NULL | 商品描述 |
| `price_cent` | `BIGINT` | NOT NULL | 商品单价，单位分 |
| `status` | `INT` | INDEX | 商品状态：1 在售，2 预售中，3 正在下架，4 已下架 |
| `is_deleted` | `TINYINT` | NOT NULL DEFAULT 0 | 逻辑删除标记 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

建议索引：

```sql
CREATE INDEX idx_products_seller_status_created ON products(seller_id, status, created_at);
CREATE INDEX idx_products_name_prefix ON products(product_name);
```

## 6. `product_inventory`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `product_id` | `BIGINT` | PK | 商品 ID |
| `available_quantity` | `INT` | NOT NULL DEFAULT 0 | 可售库存 |
| `reserved_quantity` | `INT` | NOT NULL DEFAULT 0 | 已预占库存 |
| `shipped_quantity` | `INT` | NOT NULL DEFAULT 0 | 已发货数量 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 7. `orders`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT` | PK | 订单 ID |
| `buyer_id` | `BIGINT` | INDEX | 买家用户 ID |
| `seller_id` | `BIGINT` | INDEX | 卖家用户 ID |
| `product_id` | `BIGINT` | INDEX | 商品 ID |
| `product_name_snapshot` | `VARCHAR(128)` | NOT NULL | 下单时商品名快照 |
| `unit_price_cent` | `BIGINT` | NOT NULL | 下单时单价 |
| `quantity` | `INT` | NOT NULL | 数量 |
| `total_amount_cent` | `BIGINT` | NOT NULL | 订单总金额 |
| `refund_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 退款金额 |
| `status` | `INT` | INDEX | 订单状态：1 已下单未发货，2 配送中，3 已收货，4 已退款 |
| `is_deal_completed` | `TINYINT` | NOT NULL DEFAULT 0 | 是否完成交易 |
| `paid_at` | `DATETIME(3)` | NULL | 支付时间 |
| `shipped_at` | `DATETIME(3)` | NULL | 发货时间 |
| `received_at` | `DATETIME(3)` | NULL | 收货时间 |
| `refunded_at` | `DATETIME(3)` | NULL | 退款时间 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

建议索引：

```sql
CREATE INDEX idx_orders_buyer_status_created ON orders(buyer_id, status, created_at);
CREATE INDEX idx_orders_seller_product_status ON orders(seller_id, product_id, status);
CREATE INDEX idx_orders_status_created ON orders(status, created_at);
```

## 8. `payment_transactions`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT` | PK | 流水 ID |
| `user_id` | `BIGINT` | INDEX | 用户 ID |
| `order_id` | `BIGINT` | NULL | 关联订单 |
| `type` | `INT` | INDEX | 流水类型：1 充值，2 支付扣款，3 退款入账 |
| `amount_cent` | `BIGINT` | NOT NULL | 金额，单位分 |
| `status` | `INT` | NOT NULL | 处理状态：2 成功，3 失败 |
| `idempotency_key` | `VARCHAR(128)` | UNIQUE | 幂等键 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |

## 9. `product_daily_stats`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `biz_date` | `DATE` | PK | 业务日期 |
| `product_id` | `BIGINT` | PK | 商品 ID |
| `seller_id` | `BIGINT` | INDEX | 卖家用户 ID |
| `deal_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 成交金额 |
| `refund_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 退款金额 |
| `paid_order_count` | `INT` | NOT NULL DEFAULT 0 | 支付订单数 |
| `refund_order_count` | `INT` | NOT NULL DEFAULT 0 | 退款订单数 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 10. `seller_daily_stats`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `biz_date` | `DATE` | PK | 业务日期 |
| `seller_id` | `BIGINT` | PK | 卖家用户 ID |
| `deal_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 成交金额 |
| `refund_amount_cent` | `BIGINT` | NOT NULL DEFAULT 0 | 退款金额 |
| `paid_order_count` | `INT` | NOT NULL DEFAULT 0 | 支付订单数 |
| `refund_order_count` | `INT` | NOT NULL DEFAULT 0 | 退款订单数 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |

## 11. `idempotency_records`

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `idempotency_key` | `VARCHAR(128)` | PK | 幂等键 |
| `user_id` | `BIGINT` | INDEX | 请求用户 |
| `request_path` | `VARCHAR(256)` | NOT NULL | 请求路径 |
| `request_hash` | `VARCHAR(128)` | NOT NULL | 请求参数摘要 |
| `response_code` | `INT` | NULL | 首次响应码 |
| `response_body` | `TEXT` | NULL | 首次响应体 |
| `status` | `INT` | NOT NULL | 幂等处理状态：1 处理中，2 成功，3 失败 |
| `expired_at` | `DATETIME(3)` | NOT NULL | 过期时间 |
| `created_at` | `DATETIME(3)` | NOT NULL | 创建时间 |
| `updated_at` | `DATETIME(3)` | NOT NULL | 更新时间 |
