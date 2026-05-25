# 错误处理设计

## 1. 响应格式

成功：

```json
{
  "data": {},
  "requestId": "req-uuid"
}
```

失败：

```json
{
  "code": "INVENTORY_NOT_ENOUGH",
  "message": "库存不足，请先补库存再发货",
  "requestId": "req-uuid"
}
```

## 2. HTTP 状态码

| 状态码 | 场景 |
| --- | --- |
| 200 | 查询或操作成功 |
| 400 | 参数错误 |
| 401 | 未登录或 Token 过期 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 幂等冲突或状态冲突 |
| 500 | 服务内部错误 |

## 3. 业务错误码

| 错误码 | 说明 |
| --- | --- |
| `AUTH_INVALID_CREDENTIAL` | 登录失败 |
| `AUTH_FORBIDDEN` | 无权限 |
| `VALIDATION_FAILED` | 参数校验失败 |
| `BALANCE_NOT_ENOUGH` | 余额不足 |
| `INVENTORY_NOT_ENOUGH` | 库存不足 |
| `PRODUCT_NOT_BUYABLE` | 商品不可购买 |
| `ORDER_STATUS_INVALID` | 订单状态不允许操作 |
| `PRODUCT_STATUS_INVALID` | 商品状态不允许操作 |
| `IDEMPOTENCY_CONFLICT` | 幂等键重复但请求内容不同 |
| `RESOURCE_NOT_FOUND` | 资源不存在 |

## 4. 前端展示策略

- 参数错误展示在表单字段附近。
- 余额不足提示去充值。
- 库存不足提示卖家补库存。
- 鉴权失败跳转登录页。
- 未知错误展示通用错误并附带 `requestId`。
