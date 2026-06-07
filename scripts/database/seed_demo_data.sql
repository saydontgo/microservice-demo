-- Demo seed data for local/WSL2 verification.
-- Login accounts after running this script:
--   buyer:  demo_buyer  / demo123
--   seller: demo_seller / demo123

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;
SET collation_connection = 'utf8mb4_unicode_ci';

SET @password_salt := 'microservice-demo';
SET @demo_password := 'demo123';
SET @buyer_username := 'demo_buyer' COLLATE utf8mb4_unicode_ci;
SET @seller_username := 'demo_seller' COLLATE utf8mb4_unicode_ci;
SET @buyer_hash := SHA2(CONCAT(@password_salt, ':', @demo_password), 256);
SET @seller_hash := SHA2(CONCAT(@password_salt, ':', @demo_password), 256);

START TRANSACTION;

SELECT @old_buyer_id := id FROM users WHERE username = @buyer_username LIMIT 1;
SELECT @old_seller_id := id FROM users WHERE username = @seller_username LIMIT 1;

DELETE FROM payment_transactions
WHERE user_id IN (@old_buyer_id, @old_seller_id)
   OR order_id IN (
     SELECT id FROM orders
     WHERE buyer_id = @old_buyer_id OR seller_id = @old_seller_id
   );

DELETE FROM product_daily_stats WHERE seller_id = @old_seller_id;
DELETE FROM seller_daily_stats WHERE seller_id = @old_seller_id;
DELETE FROM orders WHERE buyer_id = @old_buyer_id OR seller_id = @old_seller_id;
DELETE FROM product_inventory
WHERE product_id IN (SELECT id FROM products WHERE seller_id = @old_seller_id);
DELETE FROM products WHERE seller_id = @old_seller_id;
DELETE FROM buyer_profiles WHERE user_id = @old_buyer_id;
DELETE FROM seller_profiles WHERE user_id = @old_seller_id;
DELETE FROM users WHERE id IN (@old_buyer_id, @old_seller_id);

INSERT INTO users (username, password_hash, role, status)
VALUES
  (@buyer_username, @buyer_hash, 'BUYER', 1),
  (@seller_username, @seller_hash, 'SELLER', 1);

SELECT @buyer_id := id FROM users WHERE username = @buyer_username LIMIT 1;
SELECT @seller_id := id FROM users WHERE username = @seller_username LIMIT 1;

INSERT INTO buyer_profiles (user_id, nickname, avatar_url, phone, shipping_address, balance_cent)
VALUES (@buyer_id, 'Demo 买家', '', '13800000000', '北京市朝阳区 Demo 路 1 号', 20000000);

INSERT INTO seller_profiles (user_id, registrant_name, shop_name, shop_avatar_url, theme, total_deal_amount_cent)
VALUES (@seller_id, 'Demo 商家', 'Demo 数码旗舰店', '', 'LIGHT', 1219000);

INSERT INTO products (seller_id, product_name, description, price_cent, status)
VALUES
  (@seller_id, 'phone case classic', '经典防摔手机壳，适合买家搜索 phone 前缀', 1999, 1),
  (@seller_id, 'phone stand pro', '桌面折叠手机支架，库存充足', 3999, 1),
  (@seller_id, 'phone charger fast', '快充充电器，预售中', 8999, 2),
  (@seller_id, 'keyboard compact', '紧凑机械键盘，商家列表展示用', 23900, 1),
  (@seller_id, 'earbuds lite', '轻量蓝牙耳机，商家列表展示用', 12900, 1);

SELECT @p_case := id FROM products WHERE seller_id = @seller_id AND product_name = 'phone case classic' LIMIT 1;
SELECT @p_stand := id FROM products WHERE seller_id = @seller_id AND product_name = 'phone stand pro' LIMIT 1;
SELECT @p_charger := id FROM products WHERE seller_id = @seller_id AND product_name = 'phone charger fast' LIMIT 1;
SELECT @p_keyboard := id FROM products WHERE seller_id = @seller_id AND product_name = 'keyboard compact' LIMIT 1;
SELECT @p_earbuds := id FROM products WHERE seller_id = @seller_id AND product_name = 'earbuds lite' LIMIT 1;

INSERT INTO product_inventory (product_id, available_quantity, reserved_quantity, shipped_quantity)
VALUES
  (@p_case, 98, 1, 12),
  (@p_stand, 60, 0, 8),
  (@p_charger, 0, 0, 0),
  (@p_keyboard, 20, 0, 3),
  (@p_earbuds, 35, 0, 6);

INSERT INTO orders (
  buyer_id, seller_id, product_id, product_name_snapshot, unit_price_cent,
  quantity, total_amount_cent, refund_amount_cent, status, is_deal_completed,
  paid_at, shipped_at, received_at, refunded_at, created_at
)
VALUES
  (@buyer_id, @seller_id, @p_case, 'phone case classic', 1999, 1, 1999, 0, 1, 0, NOW(3), NULL, NULL, NULL, NOW(3) - INTERVAL 1 DAY),
  (@buyer_id, @seller_id, @p_stand, 'phone stand pro', 3999, 2, 7998, 0, 2, 0, NOW(3), NOW(3), NULL, NULL, NOW(3) - INTERVAL 2 DAY),
  (@buyer_id, @seller_id, @p_keyboard, 'keyboard compact', 23900, 1, 23900, 0, 3, 1, NOW(3), NOW(3), NOW(3), NULL, NOW(3) - INTERVAL 3 DAY),
  (@buyer_id, @seller_id, @p_earbuds, 'earbuds lite', 12900, 1, 12900, 12900, 4, 1, NOW(3), NULL, NULL, NOW(3), NOW(3) - INTERVAL 4 DAY);

SELECT @o_pending := id FROM orders WHERE buyer_id = @buyer_id AND product_id = @p_case AND status = 1 LIMIT 1;
SELECT @o_shipping := id FROM orders WHERE buyer_id = @buyer_id AND product_id = @p_stand AND status = 2 LIMIT 1;
SELECT @o_received := id FROM orders WHERE buyer_id = @buyer_id AND product_id = @p_keyboard AND status = 3 LIMIT 1;
SELECT @o_refunded := id FROM orders WHERE buyer_id = @buyer_id AND product_id = @p_earbuds AND status = 4 LIMIT 1;

INSERT INTO payment_transactions (user_id, order_id, type, amount_cent, status, idempotency_key)
VALUES
  (@buyer_id, NULL, 1, 20000000, 2, 'seed-demo-recharge'),
  (@buyer_id, @o_pending, 2, 1999, 2, 'seed-demo-pay-pending'),
  (@buyer_id, @o_shipping, 2, 7998, 2, 'seed-demo-pay-shipping'),
  (@buyer_id, @o_received, 2, 23900, 2, 'seed-demo-pay-received'),
  (@buyer_id, @o_refunded, 2, 12900, 2, 'seed-demo-pay-refunded'),
  (@buyer_id, @o_refunded, 3, 12900, 2, 'seed-demo-refund');

INSERT INTO seller_daily_stats (biz_date, seller_id, deal_amount_cent, refund_amount_cent, paid_order_count, refund_order_count)
VALUES
  (CURDATE() - INTERVAL 6 DAY, @seller_id, 120000, 0, 6, 0),
  (CURDATE() - INTERVAL 5 DAY, @seller_id, 245000, 30000, 9, 1),
  (CURDATE() - INTERVAL 4 DAY, @seller_id, 198000, 45000, 7, 1),
  (CURDATE() - INTERVAL 3 DAY, @seller_id, 365000, 64000, 11, 2),
  (CURDATE() - INTERVAL 2 DAY, @seller_id, 280000, 21000, 8, 1),
  (CURDATE() - INTERVAL 1 DAY, @seller_id, 420000, 70000, 13, 2),
  (CURDATE(), @seller_id, 310000, 15000, 10, 1);

INSERT INTO product_daily_stats (biz_date, product_id, seller_id, deal_amount_cent, refund_amount_cent, paid_order_count, refund_order_count)
VALUES
  (CURDATE() - INTERVAL 6 DAY, @p_case, @seller_id, 40000, 0, 2, 0),
  (CURDATE() - INTERVAL 5 DAY, @p_case, @seller_id, 75000, 10000, 3, 1),
  (CURDATE() - INTERVAL 4 DAY, @p_stand, @seller_id, 58000, 0, 2, 0),
  (CURDATE() - INTERVAL 3 DAY, @p_keyboard, @seller_id, 160000, 64000, 4, 2),
  (CURDATE() - INTERVAL 2 DAY, @p_earbuds, @seller_id, 80000, 21000, 3, 1),
  (CURDATE() - INTERVAL 1 DAY, @p_stand, @seller_id, 125000, 0, 5, 0),
  (CURDATE(), @p_case, @seller_id, 90000, 15000, 4, 1);

COMMIT;
