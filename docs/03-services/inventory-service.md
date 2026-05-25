# Inventory Service 设计

## 1. 职责

- 查询库存。
- 补库存。
- 下单预占库存。
- 发货扣减库存。
- 预售商品库存展示控制。

## 2. 库存字段

| 字段 | 说明 |
| --- | --- |
| `available_quantity` | 可售库存 |
| `reserved_quantity` | 已下单未发货占用量 |
| `shipped_quantity` | 已发货累计量 |

## 3. 买家展示规则

- 普通在售商品展示实际 `available_quantity`。
- 预售商品展示为 0。
- 下架中和已下架商品不展示给买家。

## 4. 补库存规则

- 卖家只能给自己的商品补库存。
- 补库存数量必须大于 0。
- `3`（正在下架） 商品允许补库存，用于完成历史订单发货。
- `2`（预售中） 商品补库存后可选择转为 `1`（在售）。

## 5. 发货库存校验

一键发货前：

```text
available_quantity >= sum(unshipped_order.quantity)
```

不满足则返回：

```text
INVENTORY_NOT_ENOUGH
```

## 6. 并发控制

库存变更必须使用行锁：

```sql
SELECT * FROM product_inventory WHERE product_id = ? FOR UPDATE;
```
