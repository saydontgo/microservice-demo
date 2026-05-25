-- 电商微服务架构 Demo MySQL 建表脚本
-- 目标 MySQL: 8.0+
-- 说明: status/type 字段统一使用 INT 枚举，不使用字符串枚举。

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '用户 ID',
  username VARCHAR(64) NOT NULL COMMENT '登录名',
  password_hash VARCHAR(255) NOT NULL COMMENT '密码摘要',
  role VARCHAR(16) NOT NULL COMMENT 'BUYER 或 SELLER',
  status INT NOT NULL DEFAULT 1 COMMENT '用户状态: 1 正常, 2 禁用',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  KEY idx_users_role_status (role, status),
  CONSTRAINT chk_users_role CHECK (role IN ('BUYER', 'SELLER')),
  CONSTRAINT chk_users_status CHECK (status IN (1, 2))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户账号表';

CREATE TABLE IF NOT EXISTS buyer_profiles (
  user_id BIGINT NOT NULL COMMENT '买家用户 ID',
  nickname VARCHAR(64) NOT NULL COMMENT '买家昵称',
  avatar_url VARCHAR(512) NULL COMMENT '买家头像 URL',
  phone VARCHAR(32) NULL COMMENT '手机号',
  shipping_address VARCHAR(512) NULL COMMENT '收货地址',
  balance_cent BIGINT NOT NULL DEFAULT 0 COMMENT '当前余额, 单位分',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (user_id),
  CONSTRAINT fk_buyer_profiles_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT chk_buyer_balance_non_negative CHECK (balance_cent >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='买家资料表';

CREATE TABLE IF NOT EXISTS seller_profiles (
  user_id BIGINT NOT NULL COMMENT '卖家用户 ID',
  registrant_name VARCHAR(64) NOT NULL COMMENT '注册人',
  shop_name VARCHAR(128) NOT NULL COMMENT '店铺名称',
  shop_avatar_url VARCHAR(512) NULL COMMENT '店铺头像 URL',
  theme VARCHAR(16) NOT NULL DEFAULT 'LIGHT' COMMENT '页面主题: LIGHT 或 DARK',
  total_deal_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '成交商品总额, 单位分',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (user_id),
  KEY idx_seller_shop_name (shop_name),
  CONSTRAINT fk_seller_profiles_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT chk_seller_theme CHECK (theme IN ('LIGHT', 'DARK')),
  CONSTRAINT chk_seller_total_deal_non_negative CHECK (total_deal_amount_cent >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='卖家资料表';

CREATE TABLE IF NOT EXISTS products (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '商品 ID',
  seller_id BIGINT NOT NULL COMMENT '卖家用户 ID',
  product_name VARCHAR(128) NOT NULL COMMENT '商品名',
  description VARCHAR(1024) NULL COMMENT '商品描述',
  price_cent BIGINT NOT NULL COMMENT '商品单价, 单位分',
  status INT NOT NULL COMMENT '商品状态: 1 在售, 2 预售中, 3 正在下架, 4 已下架',
  is_deleted TINYINT NOT NULL DEFAULT 0 COMMENT '逻辑删除标记',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_products_seller_status_created (seller_id, status, created_at),
  KEY idx_products_name_prefix (product_name),
  CONSTRAINT fk_products_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  CONSTRAINT chk_products_price_positive CHECK (price_cent > 0),
  CONSTRAINT chk_products_status CHECK (status IN (1, 2, 3, 4)),
  CONSTRAINT chk_products_is_deleted CHECK (is_deleted IN (0, 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品表';

CREATE TABLE IF NOT EXISTS product_inventory (
  product_id BIGINT NOT NULL COMMENT '商品 ID',
  available_quantity INT NOT NULL DEFAULT 0 COMMENT '可售库存',
  reserved_quantity INT NOT NULL DEFAULT 0 COMMENT '已预占库存',
  shipped_quantity INT NOT NULL DEFAULT 0 COMMENT '已发货数量',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (product_id),
  CONSTRAINT fk_product_inventory_product FOREIGN KEY (product_id) REFERENCES products(id),
  CONSTRAINT chk_inventory_available_non_negative CHECK (available_quantity >= 0),
  CONSTRAINT chk_inventory_reserved_non_negative CHECK (reserved_quantity >= 0),
  CONSTRAINT chk_inventory_shipped_non_negative CHECK (shipped_quantity >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品库存表';

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单 ID',
  buyer_id BIGINT NOT NULL COMMENT '买家用户 ID',
  seller_id BIGINT NOT NULL COMMENT '卖家用户 ID',
  product_id BIGINT NOT NULL COMMENT '商品 ID',
  product_name_snapshot VARCHAR(128) NOT NULL COMMENT '下单时商品名快照',
  unit_price_cent BIGINT NOT NULL COMMENT '下单时单价, 单位分',
  quantity INT NOT NULL COMMENT '购买数量',
  total_amount_cent BIGINT NOT NULL COMMENT '订单总金额, 单位分',
  refund_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '退款金额, 单位分',
  status INT NOT NULL COMMENT '订单状态: 1 已下单未发货, 2 配送中, 3 已收货, 4 已退款',
  is_deal_completed TINYINT NOT NULL DEFAULT 0 COMMENT '是否完成交易',
  paid_at DATETIME(3) NULL COMMENT '支付时间',
  shipped_at DATETIME(3) NULL COMMENT '发货时间',
  received_at DATETIME(3) NULL COMMENT '收货时间',
  refunded_at DATETIME(3) NULL COMMENT '退款时间',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_orders_buyer_status_created (buyer_id, status, created_at),
  KEY idx_orders_seller_product_status (seller_id, product_id, status),
  KEY idx_orders_status_created (status, created_at),
  KEY idx_orders_product_status (product_id, status),
  CONSTRAINT fk_orders_buyer FOREIGN KEY (buyer_id) REFERENCES users(id),
  CONSTRAINT fk_orders_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  CONSTRAINT fk_orders_product FOREIGN KEY (product_id) REFERENCES products(id),
  CONSTRAINT chk_orders_quantity CHECK (quantity > 0 AND quantity <= 100),
  CONSTRAINT chk_orders_unit_price_positive CHECK (unit_price_cent > 0),
  CONSTRAINT chk_orders_total_amount CHECK (total_amount_cent > 0 AND total_amount_cent <= 10000000000),
  CONSTRAINT chk_orders_refund_amount CHECK (refund_amount_cent >= 0),
  CONSTRAINT chk_orders_status CHECK (status IN (1, 2, 3, 4)),
  CONSTRAINT chk_orders_deal_completed CHECK (is_deal_completed IN (0, 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';

CREATE TABLE IF NOT EXISTS payment_transactions (
  id BIGINT NOT NULL AUTO_INCREMENT COMMENT '流水 ID',
  user_id BIGINT NOT NULL COMMENT '用户 ID',
  order_id BIGINT NULL COMMENT '关联订单 ID',
  type INT NOT NULL COMMENT '流水类型: 1 充值, 2 支付扣款, 3 退款入账',
  amount_cent BIGINT NOT NULL COMMENT '金额, 单位分',
  status INT NOT NULL COMMENT '处理状态: 2 成功, 3 失败',
  idempotency_key VARCHAR(128) NOT NULL COMMENT '幂等键',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_payment_idempotency_key (idempotency_key),
  KEY idx_payment_user_created (user_id, created_at),
  KEY idx_payment_order (order_id),
  KEY idx_payment_type_status (type, status),
  CONSTRAINT fk_payment_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_payment_order FOREIGN KEY (order_id) REFERENCES orders(id),
  CONSTRAINT chk_payment_type CHECK (type IN (1, 2, 3)),
  CONSTRAINT chk_payment_amount_positive CHECK (amount_cent > 0),
  CONSTRAINT chk_payment_status CHECK (status IN (2, 3))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付流水表';

CREATE TABLE IF NOT EXISTS product_daily_stats (
  biz_date DATE NOT NULL COMMENT '业务日期',
  product_id BIGINT NOT NULL COMMENT '商品 ID',
  seller_id BIGINT NOT NULL COMMENT '卖家用户 ID',
  deal_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '成交金额, 单位分',
  refund_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '退款金额, 单位分',
  paid_order_count INT NOT NULL DEFAULT 0 COMMENT '支付订单数',
  refund_order_count INT NOT NULL DEFAULT 0 COMMENT '退款订单数',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (biz_date, product_id),
  KEY idx_product_stats_seller_date (seller_id, biz_date),
  CONSTRAINT fk_product_stats_product FOREIGN KEY (product_id) REFERENCES products(id),
  CONSTRAINT fk_product_stats_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  CONSTRAINT chk_product_stats_amount CHECK (deal_amount_cent >= 0 AND refund_amount_cent >= 0),
  CONSTRAINT chk_product_stats_count CHECK (paid_order_count >= 0 AND refund_order_count >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品日统计表';

CREATE TABLE IF NOT EXISTS seller_daily_stats (
  biz_date DATE NOT NULL COMMENT '业务日期',
  seller_id BIGINT NOT NULL COMMENT '卖家用户 ID',
  deal_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '成交金额, 单位分',
  refund_amount_cent BIGINT NOT NULL DEFAULT 0 COMMENT '退款金额, 单位分',
  paid_order_count INT NOT NULL DEFAULT 0 COMMENT '支付订单数',
  refund_order_count INT NOT NULL DEFAULT 0 COMMENT '退款订单数',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (biz_date, seller_id),
  KEY idx_seller_stats_seller_date (seller_id, biz_date),
  CONSTRAINT fk_seller_stats_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  CONSTRAINT chk_seller_stats_amount CHECK (deal_amount_cent >= 0 AND refund_amount_cent >= 0),
  CONSTRAINT chk_seller_stats_count CHECK (paid_order_count >= 0 AND refund_order_count >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='卖家日统计表';

CREATE TABLE IF NOT EXISTS idempotency_records (
  idempotency_key VARCHAR(128) NOT NULL COMMENT '幂等键',
  user_id BIGINT NOT NULL COMMENT '请求用户 ID',
  request_path VARCHAR(256) NOT NULL COMMENT '请求路径',
  request_hash VARCHAR(128) NOT NULL COMMENT '请求参数摘要',
  response_code INT NULL COMMENT '首次响应码',
  response_body TEXT NULL COMMENT '首次响应体',
  status INT NOT NULL COMMENT '幂等处理状态: 1 处理中, 2 成功, 3 失败',
  expired_at DATETIME(3) NOT NULL COMMENT '过期时间',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (idempotency_key),
  KEY idx_idem_user_created (user_id, created_at),
  KEY idx_idem_expired_at (expired_at),
  CONSTRAINT fk_idem_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT chk_idem_status CHECK (status IN (1, 2, 3))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='幂等记录表';

SET FOREIGN_KEY_CHECKS = 1;
