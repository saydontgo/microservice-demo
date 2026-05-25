# Analytics Service 设计

## 1. 职责

- 计算卖家近 N 天成交金额趋势。
- 计算卖家近 N 天退款金额趋势。
- 计算退款率趋势。
- 为卖家商品管理列表提供商品维度指标。

## 2. 数据来源

| 表 | 用途 |
| --- | --- |
| `seller_daily_stats` | 卖家维度趋势图 |
| `product_daily_stats` | 商品维度成交和退款指标 |
| `orders` | 补偿或实时计算 |
| `payment_transactions` | 支付与退款流水校验 |

## 3. 写入时机

Demo 推荐在交易事务中同步更新统计表：

- 下单支付成功：增加成交金额和支付订单数。
- 退款成功：增加退款金额和退款订单数。

## 4. 趋势图规则

- 默认查询近 7 天。
- 支持卖家设置连续天数。
- 日期区间必须连续。
- 没有数据的日期补 0，保证前端折线图连续。

## 5. 退款率计算

```text
refund_rate = refund_amount_cent / deal_amount_cent
```

当 `deal_amount_cent = 0` 时退款率返回 0。

## 6. 缓存策略

趋势图可缓存 5 分钟：

```text
seller:trend:{seller_id}:{start}:{end}
```

当订单支付或退款成功后删除相关缓存。
