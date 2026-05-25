# 数据库分区策略

## 1. 总体原则

PRD 要求所有表默认以时间为分区，特殊说明的表按业务状态分区。Demo 设计中保留分区字段和分区 SQL 思路，实际实现时可先不启用物理分区，避免本地开发复杂度过高。

## 2. 时间分区表

以下表建议按 `created_at` 月分区：

- `users`
- `buyer_profiles`
- `seller_profiles`
- `products`
- `product_inventory`
- `payment_transactions`
- `idempotency_records`

示例：

```sql
PARTITION BY RANGE COLUMNS(created_at) (
  PARTITION p202601 VALUES LESS THAN ('2026-02-01'),
  PARTITION p202602 VALUES LESS THAN ('2026-03-01'),
  PARTITION pmax VALUES LESS THAN (MAXVALUE)
);
```

## 3. 统计表分区

`product_daily_stats` 和 `seller_daily_stats` 按 `biz_date` 月分区。

```sql
PARTITION BY RANGE COLUMNS(biz_date) (
  PARTITION p202601 VALUES LESS THAN ('2026-02-01'),
  PARTITION p202602 VALUES LESS THAN ('2026-03-01'),
  PARTITION pmax VALUES LESS THAN (MAXVALUE)
);
```

## 4. 订单表特殊分区

PRD 要求订单状态包括已下单未发货、配送中、已收货，并以订单状态为分区。`orders.status` 使用 `INT` 枚举，MySQL 可使用 LIST 分区表达状态，再在状态内结合 `created_at` 索引。

```sql
PARTITION BY LIST COLUMNS(status) (
  PARTITION p_unshipped VALUES IN (1), -- 已下单未发货
  PARTITION p_shipping VALUES IN (2),  -- 配送中
  PARTITION p_received VALUES IN (3),  -- 已收货
  PARTITION p_refunded VALUES IN (4)   -- 已退款
);
```

## 5. Demo 落地建议

- 第一阶段：先建普通表和索引，不启用物理分区。
- 第二阶段：数据量增大后添加分区 DDL。
- 文档、Repository 和查询条件提前保留 `created_at`、`biz_date`、`status`，确保后续可平滑升级。
