# 模块边界

## 1. 边界原则

- 用户身份与鉴权独立于业务资料。
- 商品资料与库存拆分，避免商品展示字段和库存交易字段耦合。
- 订单是交易事实中心，支付、库存、统计均围绕订单变化。
- 统计服务只做读模型和聚合，不直接改变订单主流程结果。

## 2. 模块依赖

| 模块 | 可依赖 | 不应依赖 |
| --- | --- | --- |
| Auth | User | Product、Order、Payment |
| User | Auth 用户 ID | Order 内部状态机 |
| Product | User 卖家资料 | Payment 余额 |
| Inventory | Product 基础信息 | Frontend 状态 |
| Order | Product、Inventory、Payment | Analytics 写模型 |
| Payment | User、Order ID | Product 展示逻辑 |
| Analytics | Order、Payment、Product 只读数据 | 写订单状态 |

## 3. 模块输出

| 模块 | 输出能力 |
| --- | --- |
| Auth | Token、当前身份、登录态校验结果 |
| User | 买家资料、卖家资料、店铺主题 |
| Product | 商品列表、商品详情、商品状态变更 |
| Inventory | 库存余量、预扣结果、补库存结果 |
| Order | 订单创建、查询、发货、确认收货、退款状态 |
| Payment | 余额、充值记录、支付流水、退款流水 |
| Analytics | 趋势图、商品维度成交和退款指标 |

## 4. 跨模块流程归属

| 流程 | 主编排模块 | 参与模块 |
| --- | --- | --- |
| 注册 | Auth | User |
| 下单支付 | Order | Product、Inventory、Payment |
| 退款 | Order | Payment、Inventory、Analytics |
| 一键发货 | Order | Inventory、Product |
| 商品下架 | Product | Order |
| 趋势图查询 | Analytics | Order、Payment |
