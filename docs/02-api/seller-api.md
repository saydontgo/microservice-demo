# Seller API 设计

## 1. 获取卖家资料

`GET /api/seller/profile`

响应：

```json
{
  "userId": 10001,
  "registrantName": "张三",
  "shopName": "张三小店",
  "shopAvatarUrl": "https://example.com/shop.png",
  "theme": "LIGHT",
  "totalDealAmountCent": 1000000
}
```

## 2. 更新卖家资料

`PUT /api/seller/profile`

请求：

```json
{
  "registrantName": "李四",
  "shopName": "李四严选",
  "shopAvatarUrl": "https://example.com/shop-new.png",
  "theme": "DARK"
}
```

## 3. 商品管理列表

`GET /api/seller/products?startDate=2026-05-18&endDate=2026-05-24&status=1&productNamePrefix=phone&shipped=false&page=1&pageSize=20`

筛选条件：

| 参数 | 说明 |
| --- | --- |
| `startDate`、`endDate` | 默认近 7 天，必须连续 |
| `status` | 商品状态编码：1 在售，2 预售中，3 正在下架，4 已下架 |
| `productId` | 精确查询商品 |
| `productNamePrefix` | 商品名前缀匹配 |
| `shipped` | 是否已发货 |

## 4. 卖家趋势图

`GET /api/seller/trends?days=7`

或：

`GET /api/seller/trends?startDate=2026-05-18&endDate=2026-05-24`

响应：

```json
{
  "points": [
    {
      "date": "2026-05-24",
      "dealAmountCent": 100000,
      "refundAmountCent": 10000,
      "refundRate": 0.1
    }
  ]
}
```

## 5. 一键发货

`POST /api/seller/products/{productId}/ship-all`

请求头：

```text
Idempotency-Key: ship-all-uuid
```

响应：

```json
{
  "productId": 30001,
  "shippedOrderCount": 10,
  "shippedQuantity": 30,
  "remainingInventory": 70
}
```

库存不足响应：

```json
{
  "code": "INVENTORY_NOT_ENOUGH",
  "message": "库存不足，请先补库存再发货"
}
```

## 6. 下架商品

`POST /api/seller/products/{productId}/delist`

响应：

```json
{
  "productId": 30001,
  "status": 3,
  "statusName": "DELISTING"
}
```
