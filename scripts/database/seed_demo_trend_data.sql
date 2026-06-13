-- Demo trend data for presentation rehearsals.
-- Date window based on the 2026-06-13 demo week:
--   this week:      2026-06-08 ~ 2026-06-14
--   next week:      2026-06-15 ~ 2026-06-21
--   week after next: 2026-06-22 ~ 2026-06-28
--
-- This script intentionally writes future-dated stats so the same database
-- already has data when next week's presentation arrives. The application
-- still rejects trend queries later than the server's current date.

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;
SET collation_connection = 'utf8mb4_unicode_ci';

SET @seller_username := 'demo_seller' COLLATE utf8mb4_unicode_ci;
SET @seller_id := (
  SELECT id FROM users WHERE username = @seller_username LIMIT 1
);

SET @p_case := (
  SELECT id FROM products WHERE seller_id = @seller_id AND product_name = 'phone case classic' LIMIT 1
);
SET @p_stand := (
  SELECT id FROM products WHERE seller_id = @seller_id AND product_name = 'phone stand pro' LIMIT 1
);
SET @p_charger := (
  SELECT id FROM products WHERE seller_id = @seller_id AND product_name = 'phone charger fast' LIMIT 1
);
SET @p_keyboard := (
  SELECT id FROM products WHERE seller_id = @seller_id AND product_name = 'keyboard compact' LIMIT 1
);
SET @p_earbuds := (
  SELECT id FROM products WHERE seller_id = @seller_id AND product_name = 'earbuds lite' LIMIT 1
);

START TRANSACTION;

INSERT INTO seller_daily_stats (
  biz_date, seller_id, deal_amount_cent, refund_amount_cent,
  paid_order_count, refund_order_count
)
SELECT
  d.biz_date, @seller_id, d.deal_amount_cent, d.refund_amount_cent,
  d.paid_order_count, d.refund_order_count
FROM (
  SELECT DATE('2026-06-08') AS biz_date, 186500 AS deal_amount_cent,  9300 AS refund_amount_cent,  8 AS paid_order_count, 1 AS refund_order_count UNION ALL
  SELECT DATE('2026-06-09'),             248900,                         0,                         11,                       0 UNION ALL
  SELECT DATE('2026-06-10'),             321600,                     28800,                         14,                       1 UNION ALL
  SELECT DATE('2026-06-11'),             274500,                     16500,                         12,                       1 UNION ALL
  SELECT DATE('2026-06-12'),             392000,                     23000,                         16,                       1 UNION ALL
  SELECT DATE('2026-06-13'),             468800,                     52000,                         19,                       2 UNION ALL
  SELECT DATE('2026-06-14'),             436200,                     18000,                         18,                       1 UNION ALL
  SELECT DATE('2026-06-15'),             512400,                     41000,                         21,                       2 UNION ALL
  SELECT DATE('2026-06-16'),             498600,                     25000,                         20,                       1 UNION ALL
  SELECT DATE('2026-06-17'),             552300,                     30000,                         23,                       1 UNION ALL
  SELECT DATE('2026-06-18'),             621700,                     57300,                         26,                       2 UNION ALL
  SELECT DATE('2026-06-19'),             704500,                     63000,                         29,                       2 UNION ALL
  SELECT DATE('2026-06-20'),             668900,                     27000,                         27,                       1 UNION ALL
  SELECT DATE('2026-06-21'),             735200,                     48000,                         30,                       2 UNION ALL
  SELECT DATE('2026-06-22'),             758400,                     39000,                         31,                       1 UNION ALL
  SELECT DATE('2026-06-23'),             801600,                     45000,                         33,                       2 UNION ALL
  SELECT DATE('2026-06-24'),             846300,                     72000,                         35,                       3 UNION ALL
  SELECT DATE('2026-06-25'),             792100,                     35000,                         32,                       1 UNION ALL
  SELECT DATE('2026-06-26'),             918500,                     86000,                         38,                       3 UNION ALL
  SELECT DATE('2026-06-27'),             955000,                     61000,                         39,                       2 UNION ALL
  SELECT DATE('2026-06-28'),            1032400,                     97000,                         42,                       3
) AS d
WHERE @seller_id IS NOT NULL
ON DUPLICATE KEY UPDATE
  deal_amount_cent = VALUES(deal_amount_cent),
  refund_amount_cent = VALUES(refund_amount_cent),
  paid_order_count = VALUES(paid_order_count),
  refund_order_count = VALUES(refund_order_count);

INSERT INTO product_daily_stats (
  biz_date, product_id, seller_id, deal_amount_cent, refund_amount_cent,
  paid_order_count, refund_order_count
)
SELECT
  d.biz_date,
  p.product_id,
  @seller_id,
  CAST(ROUND(d.deal_amount_cent * p.deal_share) AS SIGNED),
  CAST(ROUND(d.refund_amount_cent * p.refund_share) AS SIGNED),
  p.paid_order_count,
  p.refund_order_count
FROM (
  SELECT DATE('2026-06-08') AS biz_date, 186500 AS deal_amount_cent,  9300 AS refund_amount_cent UNION ALL
  SELECT DATE('2026-06-09'),             248900,                         0 UNION ALL
  SELECT DATE('2026-06-10'),             321600,                     28800 UNION ALL
  SELECT DATE('2026-06-11'),             274500,                     16500 UNION ALL
  SELECT DATE('2026-06-12'),             392000,                     23000 UNION ALL
  SELECT DATE('2026-06-13'),             468800,                     52000 UNION ALL
  SELECT DATE('2026-06-14'),             436200,                     18000 UNION ALL
  SELECT DATE('2026-06-15'),             512400,                     41000 UNION ALL
  SELECT DATE('2026-06-16'),             498600,                     25000 UNION ALL
  SELECT DATE('2026-06-17'),             552300,                     30000 UNION ALL
  SELECT DATE('2026-06-18'),             621700,                     57300 UNION ALL
  SELECT DATE('2026-06-19'),             704500,                     63000 UNION ALL
  SELECT DATE('2026-06-20'),             668900,                     27000 UNION ALL
  SELECT DATE('2026-06-21'),             735200,                     48000 UNION ALL
  SELECT DATE('2026-06-22'),             758400,                     39000 UNION ALL
  SELECT DATE('2026-06-23'),             801600,                     45000 UNION ALL
  SELECT DATE('2026-06-24'),             846300,                     72000 UNION ALL
  SELECT DATE('2026-06-25'),             792100,                     35000 UNION ALL
  SELECT DATE('2026-06-26'),             918500,                     86000 UNION ALL
  SELECT DATE('2026-06-27'),             955000,                     61000 UNION ALL
  SELECT DATE('2026-06-28'),            1032400,                     97000
) AS d
CROSS JOIN (
  SELECT @p_case AS product_id,     0.28 AS deal_share, 0.18 AS refund_share, 3 AS paid_order_count, 0 AS refund_order_count UNION ALL
  SELECT @p_stand,                  0.24,               0.15,                 3,                       0 UNION ALL
  SELECT @p_charger,                0.20,               0.08,                 2,                       0 UNION ALL
  SELECT @p_keyboard,               0.16,               0.34,                 2,                       1 UNION ALL
  SELECT @p_earbuds,                0.12,               0.25,                 2,                       1
) AS p
WHERE @seller_id IS NOT NULL
  AND p.product_id IS NOT NULL
ON DUPLICATE KEY UPDATE
  deal_amount_cent = VALUES(deal_amount_cent),
  refund_amount_cent = VALUES(refund_amount_cent),
  paid_order_count = VALUES(paid_order_count),
  refund_order_count = VALUES(refund_order_count);

COMMIT;
