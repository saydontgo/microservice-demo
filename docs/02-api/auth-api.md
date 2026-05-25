# Auth API 设计

## 1. 通用约定

- Base Path：`/api/auth`
- 写接口需要 `Idempotency-Key` 请求头。
- 登录成功返回 Bearer Token。

## 2. 注册

`POST /api/auth/register`

请求：

```json
{
  "username": "seller001",
  "password": "plain-password",
  "role": "SELLER",
  "profile": {
    "registrantName": "张三",
    "shopName": "张三小店"
  }
}
```

买家注册时 `profile` 包含 `nickname`、`phone`、`shippingAddress`。

响应：

```json
{
  "userId": 10001,
  "role": "SELLER"
}
```

## 3. 登录

`POST /api/auth/login`

请求：

```json
{
  "username": "buyer001",
  "password": "plain-password",
  "role": "BUYER"
}
```

响应：

```json
{
  "token": "jwt-or-random-token",
  "expiresIn": 7200,
  "role": "BUYER"
}
```

## 4. 退出登录

`POST /api/auth/logout`

请求头：

```text
Authorization: Bearer <token>
```

响应：

```json
{
  "success": true
}
```

## 5. 错误码

| 错误码 | 说明 |
| --- | --- |
| `AUTH_INVALID_CREDENTIAL` | 用户名、密码或身份错误 |
| `AUTH_ROLE_MISMATCH` | 登录身份与账号身份不一致 |
| `AUTH_TOKEN_EXPIRED` | Token 过期 |
| `AUTH_FORBIDDEN` | 无权限访问接口 |
