# 核心取数 SQL

本文 SQL 中所有 `status`、`type` 条件均使用 `INT` 枚举编码。核心编码：商品状态 1 在售、2 预售中、3 正在下架、4 已下架；订单状态 1 已下单未发货、2 配送中、3 已收货、4 已退款。

## 1. 卖家商品管理列表

```sql
SELECT
  p.id AS product_id,
  p.product_name,
  p.status,
  p.price_cent,
  CASE WHEN p.status = 2 THEN 0 ELSE i.available_quantity END AS display_inventory,
  COALESCE(SUM(s.deal_amount_cent), 0) AS deal_amount_cent,
  COALESCE(SUM(s.refund_amount_cent), 0) AS refund_amount_cent,
  CASE
    WHEN COALESCE(SUM(s.deal_amount_cent), 0) = 0 THEN 0
    ELSE COALESCE(SUM(s.refund_amount_cent), 0) / COALESCE(SUM(s.deal_amount_cent), 0)
  END AS refund_rate
FROM products p
JOIN product_inventory i ON i.product_id = p.id
LEFT JOIN product_daily_stats s
  ON s.product_id = p.id
 AND s.biz_date BETWEEN ? AND ?
WHERE p.seller_id = ?
  AND (? IS NULL OR p.status = ?)
  AND (? IS NULL OR p.id = ?)
  AND (? IS NULL OR p.product_name LIKE CONCAT(?, '%'))
  AND (
    ? IS NULL
    OR (? = TRUE AND EXISTS (
      SELECT 1
      FROM orders o
      WHERE o.product_id = p.id
        AND o.seller_id = p.seller_id
        AND o.status IN (2, 3)
    ))
    OR (? = FALSE AND EXISTS (
      SELECT 1
      FROM orders o
      WHERE o.product_id = p.id
        AND o.seller_id = p.seller_id
        AND o.status = 1
    ))
  )
GROUP BY p.id, p.product_name, p.status, p.price_cent, i.available_quantity
ORDER BY p.created_at DESC
LIMIT ? OFFSET ?;
```

## 2. 卖家商品是否有待发货订单

```sql
SELECT COUNT(*) AS unshipped_order_count,
       COALESCE(SUM(quantity), 0) AS unshipped_quantity
FROM orders
WHERE seller_id = ?
  AND product_id = ?
  AND status = 1;
```

## 3. 卖家趋势图

```sql
SELECT
  biz_date,
  deal_amount_cent,
  refund_amount_cent,
  CASE
    WHEN deal_amount_cent = 0 THEN 0
    ELSE refund_amount_cent / deal_amount_cent
  END AS refund_rate
FROM seller_daily_stats
WHERE seller_id = ?
  AND biz_date BETWEEN ? AND ?
ORDER BY biz_date ASC;
```

## 4. 买家商品名前缀搜索

```sql
SELECT
  p.id AS product_id,
  p.product_name,
  p.price_cent,
  p.status,
  CASE WHEN p.status = 2 THEN 0 ELSE i.available_quantity END AS display_inventory,
  sp.shop_name
FROM products p
JOIN product_inventory i ON i.product_id = p.id
JOIN seller_profiles sp ON sp.user_id = p.seller_id
WHERE p.product_name LIKE CONCAT(?, '%')
  AND p.status IN (1, 2)
  AND p.is_deleted = 0
ORDER BY p.created_at DESC
LIMIT ? OFFSET ?;
```

## 5. 买家订单列表默认查询

```sql
SELECT
  id,
  product_id,
  product_name_snapshot,
  quantity,
  total_amount_cent,
  status,
  created_at,
  shipped_at
FROM orders
WHERE buyer_id = ?
  AND status IN (1, 2)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

## 6. 买家按状态查询订单

```sql
SELECT
  id,
  product_id,
  product_name_snapshot,
  quantity,
  total_amount_cent,
  refund_amount_cent,
  status,
  created_at,
  updated_at
FROM orders
WHERE buyer_id = ?
  AND (? IS NULL OR status = ?)
  AND created_at BETWEEN ? AND ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

## 7. 下单前余额校验

```sql
SELECT balance_cent
FROM buyer_profiles
WHERE user_id = ?
FOR UPDATE;
```

## 8. 下单前库存校验

```sql
SELECT available_quantity
FROM product_inventory
WHERE product_id = ?
FOR UPDATE;
```

## 9. 商品是否可转已下架

```sql
SELECT COUNT(*) AS unfinished_count
FROM orders
WHERE product_id = ?
  AND status IN (1, 2);
```

## 10. 卖家成交总额

```sql
SELECT total_deal_amount_cent
FROM seller_profiles
WHERE user_id = ?;
```
