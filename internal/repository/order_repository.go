package repository

import (
	"context"
	"database/sql"
	"time"

	orderdomain "microservice-demo/internal/domain/order"
	productdomain "microservice-demo/internal/domain/product"
)

type OrderRepository struct {
	db *sql.DB
}

type CreateOrderParams struct {
	BuyerID        int64
	ProductID      int64
	Quantity       int
	IdempotencyKey string
}

type Order struct {
	ID                  int64
	ProductID           int64
	ProductNameSnapshot string
	Quantity            int
	TotalAmountCent     int64
	RefundAmountCent    int64
	Status              int
	CreatedAt           time.Time
	ShippedAt           sql.NullTime
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, params CreateOrderParams) (int64, int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var sellerID int64
	var productName string
	var unitPriceCent int64
	var productStatus int
	const productQuery = `
SELECT seller_id, product_name, price_cent, status
FROM products
WHERE id = ? AND is_deleted = 0
FOR UPDATE`
	if err := tx.QueryRowContext(ctx, productQuery, params.ProductID).Scan(&sellerID, &productName, &unitPriceCent, &productStatus); err != nil {
		return 0, 0, err
	}
	if productStatus != productdomain.StatusOnSale && productStatus != productdomain.StatusPreSale {
		return 0, 0, sql.ErrNoRows
	}

	totalAmount := unitPriceCent * int64(params.Quantity)
	var balance int64
	if err := tx.QueryRowContext(ctx, `SELECT balance_cent FROM buyer_profiles WHERE user_id = ? FOR UPDATE`, params.BuyerID).Scan(&balance); err != nil {
		return 0, 0, err
	}
	if balance < totalAmount {
		return 0, balance, ErrBalanceNotEnough
	}

	var available int
	if err := tx.QueryRowContext(ctx, `SELECT available_quantity FROM product_inventory WHERE product_id = ? FOR UPDATE`, params.ProductID).Scan(&available); err != nil {
		return 0, 0, err
	}
	if productStatus == productdomain.StatusOnSale && available < params.Quantity {
		return 0, balance, ErrInventoryNotEnough
	}

	if _, err := tx.ExecContext(ctx, `UPDATE buyer_profiles SET balance_cent = balance_cent - ? WHERE user_id = ?`, totalAmount, params.BuyerID); err != nil {
		return 0, 0, err
	}
	if productStatus == productdomain.StatusOnSale {
		const updateInventory = `
UPDATE product_inventory
SET available_quantity = available_quantity - ?, reserved_quantity = reserved_quantity + ?
WHERE product_id = ?`
		if _, err := tx.ExecContext(ctx, updateInventory, params.Quantity, params.Quantity, params.ProductID); err != nil {
			return 0, 0, err
		}
	}

	const insertOrder = `
INSERT INTO orders (buyer_id, seller_id, product_id, product_name_snapshot, unit_price_cent, quantity, total_amount_cent, status, paid_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP(3))`
	result, err := tx.ExecContext(ctx, insertOrder, params.BuyerID, sellerID, params.ProductID, productName, unitPriceCent, params.Quantity, totalAmount, orderdomain.StatusPlacedUnshipped)
	if err != nil {
		return 0, 0, err
	}
	orderID, err := result.LastInsertId()
	if err != nil {
		return 0, 0, err
	}

	const insertPayment = `
INSERT INTO payment_transactions (user_id, order_id, type, amount_cent, status, idempotency_key)
VALUES (?, ?, 2, ?, 2, ?)`
	if _, err := tx.ExecContext(ctx, insertPayment, params.BuyerID, orderID, totalAmount, params.IdempotencyKey); err != nil {
		return 0, 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE seller_profiles SET total_deal_amount_cent = total_deal_amount_cent + ? WHERE user_id = ?`, totalAmount, sellerID); err != nil {
		return 0, 0, err
	}

	bizDate := time.Now().Format("2006-01-02")
	const upsertProductStats = `
INSERT INTO product_daily_stats (biz_date, product_id, seller_id, deal_amount_cent, paid_order_count)
VALUES (?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE deal_amount_cent = deal_amount_cent + VALUES(deal_amount_cent), paid_order_count = paid_order_count + 1`
	if _, err := tx.ExecContext(ctx, upsertProductStats, bizDate, params.ProductID, sellerID, totalAmount); err != nil {
		return 0, 0, err
	}
	const upsertSellerStats = `
INSERT INTO seller_daily_stats (biz_date, seller_id, deal_amount_cent, paid_order_count)
VALUES (?, ?, ?, 1)
ON DUPLICATE KEY UPDATE deal_amount_cent = deal_amount_cent + VALUES(deal_amount_cent), paid_order_count = paid_order_count + 1`
	if _, err := tx.ExecContext(ctx, upsertSellerStats, bizDate, sellerID, totalAmount); err != nil {
		return 0, 0, err
	}

	return orderID, balance - totalAmount, tx.Commit()
}

func (r *OrderRepository) ListBuyerOrders(ctx context.Context, buyerID int64, statuses []int, limit, offset int) ([]Order, error) {
	query := `
SELECT id, product_id, product_name_snapshot, quantity, total_amount_cent, refund_amount_cent, status, created_at, shipped_at
FROM orders
WHERE buyer_id = ? AND status IN (?, ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, buyerID, statuses[0], statuses[1], limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]Order, 0)
	for rows.Next() {
		var item Order
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductNameSnapshot, &item.Quantity, &item.TotalAmountCent, &item.RefundAmountCent, &item.Status, &item.CreatedAt, &item.ShippedAt); err != nil {
			return nil, err
		}
		orders = append(orders, item)
	}
	return orders, rows.Err()
}
