#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8000}"

# 假设 token 已经存在，也可以在执行前 export
# BUYER_TOKEN=$(curl -s -X POST http://127.0.0.1:8000/api/auth/login \
#   -H 'Content-Type: application/json' \
#   -d '{"username":"buyer001","password":"123456","role":"BUYER"}' \
#   | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# SELLER_TOKEN=$(curl -s -X POST http://127.0.0.1:8000/api/auth/login \
#   -H 'Content-Type: application/json' \
#   -d '{"username":"seller001","password":"123456","role":"SELLER"}' \
#   | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
BUYER_TOKEN='eyJ1c2VySWQiOjUsInJvbGUiOiJCVVlFUiIsImV4cGlyZXNBdCI6MTc4MDIyNTAzOH0.dw2-Drn0mSUptF4dHgnPVJRQVrMB0lMy5uuFflQ4OxE'
SELLER_TOKEN='eyJ1c2VySWQiOjYsInJvbGUiOiJTRUxMRVIiLCJleHBpcmVzQXQiOjE3ODAyMjUwNTB9.3NOfo-VOza_MVc6ZKd58T-saA_HvpPUPFXAVS9Ux6HY'

if [ -z "${SELLER_TOKEN:-}" ] || [ -z "${BUYER_TOKEN:-}" ]; then
  echo "请先设置 SELLER_TOKEN 和 BUYER_TOKEN"
  exit 1
fi

echo "BASE_URL=${BASE_URL}"

echo
echo "========== 场景 A：没有进行中订单，下架应直接变成 OFF_SHELF(4) =========="

PRODUCT_A_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/seller/products" \
    -H "Authorization: Bearer ${SELLER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
      "productName": "delist-direct-test",
      "description": "没有订单，点击下架应直接已下架",
      "priceCent": 1000,
      "status": 1,
      "initialInventory": 10
    }'
)

echo "创建商品 A 响应："
echo "${PRODUCT_A_RESP}" | jq .

PRODUCT_A_ID=$(echo "${PRODUCT_A_RESP}" | jq -r '.data.productId')

DELIST_A_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/seller/products/${PRODUCT_A_ID}/delist" \
    -H "Authorization: Bearer ${SELLER_TOKEN}"
)

echo "商品 A 下架响应："
echo "${DELIST_A_RESP}" | jq .

STATUS_A=$(echo "${DELIST_A_RESP}" | jq -r '.data.status')
STATUS_NAME_A=$(echo "${DELIST_A_RESP}" | jq -r '.data.statusName')

if [ "${STATUS_A}" != "4" ]; then
  echo "场景 A 失败：期望 status=4，实际 status=${STATUS_A}, statusName=${STATUS_NAME_A}"
  exit 1
fi

echo "场景 A 通过：无进行中订单时直接 OFF_SHELF(4)"

echo
echo "========== 场景 B：有进行中订单，下架先 DELISTING(3)，最后订单完成后自动 OFF_SHELF(4) =========="

PRODUCT_B_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/seller/products" \
    -H "Authorization: Bearer ${SELLER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
      "productName": "delist-after-order-complete-test",
      "description": "有订单时先正在下架，最后订单完成后自动已下架",
      "priceCent": 1000,
      "status": 1,
      "initialInventory": 10
    }'
)

echo "创建商品 B 响应："
echo "${PRODUCT_B_RESP}" | jq .

PRODUCT_B_ID=$(echo "${PRODUCT_B_RESP}" | jq -r '.data.productId')

ORDER_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/buyer/orders" \
    -H "Authorization: Bearer ${BUYER_TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: order-delist-${PRODUCT_B_ID}-$(date +%s%N)" \
    -d "{
      \"productId\": ${PRODUCT_B_ID},
      \"quantity\": 1
    }"
)

echo "买家下单响应："
echo "${ORDER_RESP}" | jq .

ORDER_ID=$(echo "${ORDER_RESP}" | jq -r '.data.orderId')

DELIST_B_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/seller/products/${PRODUCT_B_ID}/delist" \
    -H "Authorization: Bearer ${SELLER_TOKEN}"
)

echo "商品 B 有进行中订单时下架响应："
echo "${DELIST_B_RESP}" | jq .

STATUS_B=$(echo "${DELIST_B_RESP}" | jq -r '.data.status')
STATUS_NAME_B=$(echo "${DELIST_B_RESP}" | jq -r '.data.statusName')

if [ "${STATUS_B}" != "3" ]; then
  echo "场景 B 第一步失败：期望 status=3，实际 status=${STATUS_B}, statusName=${STATUS_NAME_B}"
  exit 1
fi

echo "商品 B 已进入 DELISTING(3)"

SHIP_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/seller/products/${PRODUCT_B_ID}/ship-all" \
    -H "Authorization: Bearer ${SELLER_TOKEN}"
)

echo "商家发货响应："
echo "${SHIP_RESP}" | jq .

RECEIVE_RESP=$(
  curl -sS -X POST "${BASE_URL}/api/buyer/orders/${ORDER_ID}/receive" \
    -H "Authorization: Bearer ${BUYER_TOKEN}"
)

echo "买家确认收货响应："
echo "${RECEIVE_RESP}" | jq .

SELLER_PRODUCTS_RESP=$(
  curl -sS -X GET "${BASE_URL}/api/seller/products?productId=${PRODUCT_B_ID}&page=1&pageSize=20" \
    -H "Authorization: Bearer ${SELLER_TOKEN}"
)

echo "确认收货后查询商品 B："
echo "${SELLER_PRODUCTS_RESP}" | jq .

FINAL_STATUS_B=$(echo "${SELLER_PRODUCTS_RESP}" | jq -r '.data.items[0].status')
FINAL_STATUS_NAME_B=$(echo "${SELLER_PRODUCTS_RESP}" | jq -r '.data.items[0].statusName')

if [ "${FINAL_STATUS_B}" != "4" ]; then
  echo "场景 B 失败：期望最终 status=4，实际 status=${FINAL_STATUS_B}, statusName=${FINAL_STATUS_NAME_B}"
  exit 1
fi

echo "场景 B 通过：最后一个订单确认收货后，商品自动 OFF_SHELF(4)"

echo
echo "全部测试通过"