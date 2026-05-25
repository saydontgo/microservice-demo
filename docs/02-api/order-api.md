# Order API 设计

## 1. 订单状态机

```text
PLACED_UNSHIPPED -> SHIPPING -> RECEIVED
PLACED_UNSHIPPED -> REFUNDED
```

## 2. 创建订单

入口由买家 API 暴露：`POST /api/buyer/orders`。

服务内部步骤：

1. 校验商品存在且买家可见。
2. 校验购买数量和金额上限。
3. 锁定买家余额。
4. 锁定库存。
5. 扣减余额并预占库存。
6. 创建订单和支付流水。
7. 更新卖家成交额和统计表。

## 3. 发货

入口由卖家 API 暴露：`POST /api/seller/products/{productId}/ship-all`。

服务内部步骤：

1. 校验商品归属当前卖家。
2. 聚合未发货订单数量。
3. 锁定库存行。
4. 库存不足则返回错误。
5. 扣减可售库存和预占库存。
6. 批量更新订单为 `2`（配送中）。

## 4. 确认收货

入口：`POST /api/buyer/orders/{orderId}/receive`。

规则：

- 仅订单所属买家可操作。
- 仅 `2`（配送中） 状态可确认收货。
- 确认后状态为 `3`（已收货）。
- `is_deal_completed` 设置为 1。

## 5. 退款

入口：`POST /api/buyer/orders/{orderId}/refund`。

规则：

- 仅订单所属买家可操作。
- 仅 `1`（已下单未发货） 状态可退款。
- 退款成功后买家余额增加。
- 卖家成交总额减少。
- 统计表增加退款金额。
- 订单状态为 `4`（已退款）。

## 6. 订单查询默认规则

买家订单管理页默认查询：

```text
status IN (1, 2)
```

卖家发货逻辑查询：

```text
status = 1
```
