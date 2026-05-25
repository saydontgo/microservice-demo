# 前端架构设计

## 1. 技术边界

PRD 要求前端使用前端三件套，即 HTML、CSS、JavaScript。当前阶段只写设计文档，不生成前端代码。

## 2. 页面结构

```text
Login Page
  |-- Role Select: BUYER / SELLER
  |-- Login Form
  |-- Register Form

Buyer App
  |-- Product Browse Tab
  |-- Order Management Tab
  |-- Settings Tab

Seller App
  |-- Product Management Tab
  |-- Settings Tab
```

## 3. 状态管理

前端需要维护：

| 状态 | 说明 |
| --- | --- |
| `token` | 登录凭证 |
| `role` | 当前身份 |
| `profile` | 当前用户资料 |
| `theme` | 当前主题 |
| `filters` | 表格筛选条件 |
| `pagination` | 分页信息 |
| `loading` | 请求状态 |
| `error` | 错误提示 |

## 4. API 调用约定

- 所有业务请求携带 `Authorization: Bearer <token>`。
- 写请求生成 `Idempotency-Key`。
- 金额统一由后端返回分，前端展示为元。
- 错误码按统一弹窗或表格内提示处理。

## 5. UI 一致性

买家商品浏览、卖家商品管理、买家订单管理都使用一致的表格检索样式：

- 顶部筛选区。
- 中间表格区。
- 右侧或行内操作按钮。
- 底部分页。
- 空状态和错误状态。
