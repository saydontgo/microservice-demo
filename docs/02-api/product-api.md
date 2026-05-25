# Product API 设计

## 1. 创建商品

`POST /api/seller/products`

请求头：

```text
Idempotency-Key: create-product-uuid
```

请求：

```json
{
  "productName": "phone case",
  "description": "透明手机壳",
  "priceCent": 1999,
  "status": 1,
  "initialInventory": 100
}
```

`status` 使用 `INT` 枚举：1 在售，2 预售中，3 正在下架，4 已下架。请求体不传字符串状态名。

## 2. 更新商品

`PUT /api/seller/products/{productId}`

允许更新：

- 商品名。
- 描述。
- 价格。
- 状态，状态流转需符合规则。

## 3. 补库存

`POST /api/seller/products/{productId}/inventory/add`

请求头：

```text
Idempotency-Key: add-inventory-uuid
```

请求：

```json
{
  "quantity": 100
}
```

响应：

```json
{
  "productId": 30001,
  "availableQuantity": 200
}
```

## 4. 商品状态流转

| 当前状态 | 目标状态 | 触发动作 | 说明 |
| --- | --- | --- | --- |
| `2`（预售中） | `1`（在售） | 卖家上架 | 需要库存大于 0 |
| `1`（在售） | `3`（正在下架） | 卖家下架 | 买家侧立即不可见 |
| `3`（正在下架） | `4`（已下架） | 系统或卖家触发 | 无未完成订单 |
| `4`（已下架） | `1`（在售） | 卖家重新上架 | 需要库存大于 0 |

## 5. 前缀搜索规则

- 只支持商品名前缀匹配。
- 买家侧只返回 `1`（在售） 和 `2`（预售中）。
- 卖家侧返回自身商品，不受买家可见性限制。
