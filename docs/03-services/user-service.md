# User Service 设计

## 1. 职责

- 买家资料查询和更新。
- 卖家资料查询和更新。
- 买家余额查询展示。
- 卖家主题配置。
- 卖家成交总额展示。

## 2. 买家资料

字段来源：`buyer_profiles`。

允许用户更新：

- `nickname`
- `avatar_url`
- `phone`
- `shipping_address`

不允许前端直接更新：

- `balance_cent`

## 3. 卖家资料

字段来源：`seller_profiles`。

允许用户更新：

- `registrant_name`
- `shop_name`
- `shop_avatar_url`
- `theme`

不允许前端直接更新：

- `total_deal_amount_cent`

## 4. 缓存策略

- 查询资料时可读取 Redis `user:profile:{role}:{user_id}`。
- 更新资料后删除对应缓存。
- 余额和成交总额可展示在资料接口中，但以 MySQL 查询结果为准。

## 5. 校验规则

| 字段 | 规则 |
| --- | --- |
| 手机号 | 可为空；非空时满足基本格式 |
| 头像 URL | 可为空；非空时限制长度 512 |
| 店铺名称 | 非空，长度不超过 128 |
| 主题 | 仅允许 `LIGHT`、`DARK` |
