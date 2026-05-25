# Redis 缓存设计

## 1. 使用原则

Redis 只存放可重建数据、登录态和幂等控制信息。余额、库存、订单状态等强一致数据以 MySQL 为准。

## 2. Key 设计

| Key | 类型 | TTL | 说明 |
| --- | --- | --- | --- |
| `auth:token:{token}` | String | 2h | Token 到用户信息映射 |
| `user:profile:{role}:{user_id}` | String JSON | 10m | 用户资料缓存 |
| `product:detail:{product_id}` | String JSON | 5m | 商品详情缓存 |
| `product:search:{prefix}:{status}` | List/String JSON | 1m | 商品名前缀搜索缓存，`status` 使用商品状态 INT 编码 |
| `seller:trend:{seller_id}:{start}:{end}` | String JSON | 5m | 卖家趋势图缓存 |
| `idem:{user_id}:{idempotency_key}` | String JSON | 24h | 幂等请求结果 |
| `rate:login:{username}` | String/Counter | 10m | 登录失败计数 |

## 3. 缓存更新策略

| 场景 | 策略 |
| --- | --- |
| 商品资料更新 | 删除 `product:detail:{product_id}` 和相关搜索缓存 |
| 商品状态变更 | 删除商品详情、搜索缓存、卖家商品列表缓存 |
| 用户资料更新 | 删除用户资料缓存 |
| 订单支付/退款/确认收货 | 删除卖家趋势缓存和商品统计缓存 |
| 充值 | 不缓存余额，直接查 MySQL |

## 4. 幂等控制

写操作先尝试写入 Redis 幂等 Key：

```text
SET idem:{user_id}:{key} PROCESSING NX EX 86400
```

- 成功：进入业务处理。
- 失败：读取已有结果并返回，避免重复扣款、退款、发货。
- 业务成功后将 Key 内容更新为响应摘要。
- MySQL 的 `idempotency_records` 作为 Redis 丢失后的兜底。

## 5. 不建议缓存的数据

- 买家余额。
- 商品库存交易态。
- 订单状态。
- 支付流水。

这些数据一旦缓存，容易产生一致性问题；Demo 阶段直接读 MySQL 更清晰。
