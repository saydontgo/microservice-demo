# Buyer API 设计

## 1. 获取买家资料

`GET /api/buyer/profile`

响应：

```json
{
  "userId": 20001,
  "nickname": "买家A",
  "avatarUrl": "https://example.com/a.png",
  "phone": "13800000000",
  "shippingAddress": "北京市朝阳区",
  "balanceCent": 100000
}
```

## 2. 更新买家资料

`PUT /api/buyer/profile`

请求：

```json
{
  "nickname": "新昵称",
  "avatarUrl": "https://example.com/new.png",
  "phone": "13900000000",
  "shippingAddress": "上海市浦东新区"
}
```

## 3. 充值

`POST /api/buyer/balance/recharge`

请求头：

```text
Idempotency-Key: recharge-uuid
```

请求：

```json
{
  "amountCent": 500000
}
```

响应：

```json
{
  "balanceCent": 600000,
  "transactionId": 90001
}
```

## 4. 商品搜索

`GET /api/buyer/products?namePrefix=phone&page=1&pageSize=20`

响应：

```json
{
  "items": [
    {
      "productId": 30001,
      "productName": "phone case",
      "priceCent": 1999,
      "status": 1,
      "statusName": "ON_SALE",
      "displayInventory": 100,
      "shopName": "数码店"
    }
  ],
  "total": 1
}
```

## 5. 下单并支付

`POST /api/buyer/orders`

请求头：

```text
Idempotency-Key: order-uuid
```

请求：

```json
{
  "productId": 30001,
  "quantity": 2
}
```

响应：

```json
{
  "orderId": 50001,
  "status": 1,
  "statusName": "PLACED_UNSHIPPED",
  "totalAmountCent": 3998,
  "balanceCent": 96002
}
```

## 6. 查询订单

`GET /api/buyer/orders?status=2&page=1&pageSize=20`

未传 `status` 时默认返回 `1`（已下单未发货） 和 `2`（配送中）。

## 7. 确认收货

`POST /api/buyer/orders/{orderId}/receive`

请求头：

```text
Idempotency-Key: receive-uuid
```

## 8. 退款

`POST /api/buyer/orders/{orderId}/refund`

仅 `1`（已下单未发货） 订单允许退款。

请求头：

```text
Idempotency-Key: refund-uuid
```
