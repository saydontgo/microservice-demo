-- MySQL 冒烟测试
-- 运行前请先执行 scripts/database/create_tables.sql。

DELIMITER $$

DROP PROCEDURE IF EXISTS run_microservice_demo_mysql_smoke_test $$

CREATE PROCEDURE run_microservice_demo_mysql_smoke_test()
BEGIN
  DECLARE v_table_count INT DEFAULT 0;
  DECLARE v_bad_enum_columns INT DEFAULT 0;
  DECLARE v_seller_id BIGINT DEFAULT 0;
  DECLARE v_buyer_id BIGINT DEFAULT 0;
  DECLARE v_product_id BIGINT DEFAULT 0;
  DECLARE v_order_id BIGINT DEFAULT 0;
  DECLARE v_unfinished_count INT DEFAULT 0;
  DECLARE v_payment_count INT DEFAULT 0;
  DECLARE v_idem_count INT DEFAULT 0;
  DECLARE v_suffix VARCHAR(32) DEFAULT REPLACE(UUID(), '-', '');

  SELECT COUNT(*) INTO v_table_count
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name IN (
      'users', 'buyer_profiles', 'seller_profiles', 'products', 'product_inventory',
      'orders', 'payment_transactions', 'product_daily_stats', 'seller_daily_stats',
      'idempotency_records'
    );

  IF v_table_count <> 10 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'FAIL: expected 10 demo tables';
  END IF;

  SELECT COUNT(*) INTO v_bad_enum_columns
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND (
      (table_name = 'users' AND column_name = 'status' AND data_type <> 'int')
      OR (table_name = 'products' AND column_name = 'status' AND data_type <> 'int')
      OR (table_name = 'orders' AND column_name = 'status' AND data_type <> 'int')
      OR (table_name = 'payment_transactions' AND column_name IN ('type', 'status') AND data_type <> 'int')
      OR (table_name = 'idempotency_records' AND column_name = 'status' AND data_type <> 'int')
    );

  IF v_bad_enum_columns <> 0 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'FAIL: status/type columns must be INT';
  END IF;

  START TRANSACTION;

  INSERT INTO users(username, password_hash, role, status)
  VALUES (CONCAT('smoke_seller_', v_suffix), 'hash_for_test', 'SELLER', 1);
  SET v_seller_id = LAST_INSERT_ID();

  INSERT INTO seller_profiles(user_id, registrant_name, shop_name, shop_avatar_url, theme, total_deal_amount_cent)
  VALUES (v_seller_id, '测试注册人', CONCAT('smoke_shop_', v_suffix), NULL, 'LIGHT', 0);

  INSERT INTO users(username, password_hash, role, status)
  VALUES (CONCAT('smoke_buyer_', v_suffix), 'hash_for_test', 'BUYER', 1);
  SET v_buyer_id = LAST_INSERT_ID();

  INSERT INTO buyer_profiles(user_id, nickname, avatar_url, phone, shipping_address, balance_cent)
  VALUES (v_buyer_id, CONCAT('smoke_buyer_', v_suffix), NULL, '13800000000', '测试地址', 1000000);

  INSERT INTO products(seller_id, product_name, description, price_cent, status, is_deleted)
  VALUES (v_seller_id, CONCAT('smoke_product_', v_suffix), '测试商品', 1999, 1, 0);
  SET v_product_id = LAST_INSERT_ID();

  INSERT INTO product_inventory(product_id, available_quantity, reserved_quantity, shipped_quantity)
  VALUES (v_product_id, 100, 0, 0);

  INSERT INTO orders(
    buyer_id, seller_id, product_id, product_name_snapshot, unit_price_cent, quantity,
    total_amount_cent, refund_amount_cent, status, is_deal_completed, paid_at
  ) VALUES (
    v_buyer_id, v_seller_id, v_product_id, CONCAT('smoke_product_', v_suffix), 1999, 2,
    3998, 0, 1, 0, NOW(3)
  );
  SET v_order_id = LAST_INSERT_ID();

  INSERT INTO payment_transactions(user_id, order_id, type, amount_cent, status, idempotency_key)
  VALUES (v_buyer_id, v_order_id, 2, 3998, 2, CONCAT('smoke_pay_', v_suffix));

  INSERT INTO product_daily_stats(
    biz_date, product_id, seller_id, deal_amount_cent, refund_amount_cent, paid_order_count, refund_order_count
  ) VALUES (CURRENT_DATE(), v_product_id, v_seller_id, 3998, 0, 1, 0);

  INSERT INTO seller_daily_stats(
    biz_date, seller_id, deal_amount_cent, refund_amount_cent, paid_order_count, refund_order_count
  ) VALUES (CURRENT_DATE(), v_seller_id, 3998, 0, 1, 0);

  INSERT INTO idempotency_records(
    idempotency_key, user_id, request_path, request_hash, response_code, response_body, status, expired_at
  ) VALUES (
    CONCAT('smoke_idem_', v_suffix), v_buyer_id, '/api/buyer/orders', 'hash_for_test', 200, '{"ok":true}', 2, DATE_ADD(NOW(3), INTERVAL 1 DAY)
  );

  SELECT COUNT(*) INTO v_unfinished_count
  FROM orders
  WHERE buyer_id = v_buyer_id
    AND status IN (1, 2);

  IF v_unfinished_count <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'FAIL: unfinished order query failed';
  END IF;

  SELECT COUNT(*) INTO v_payment_count
  FROM payment_transactions
  WHERE order_id = v_order_id
    AND type = 2
    AND status = 2;

  IF v_payment_count <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'FAIL: payment transaction enum query failed';
  END IF;

  SELECT COUNT(*) INTO v_idem_count
  FROM idempotency_records
  WHERE user_id = v_buyer_id
    AND status = 2;

  IF v_idem_count <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'FAIL: idempotency record query failed';
  END IF;

  ROLLBACK;

  SELECT 'PASS: MySQL schema and smoke data checks passed' AS result;
END $$

DELIMITER ;

CALL run_microservice_demo_mysql_smoke_test();
DROP PROCEDURE IF EXISTS run_microservice_demo_mysql_smoke_test;
