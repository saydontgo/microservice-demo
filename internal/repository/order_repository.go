package repository

import (
	"context"
	"database/sql"
	"strings"
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
	if params.IdempotencyKey != "" {
		orderID, balance, found, err := r.findExistingOrderPayment(ctx, params)
		if err != nil {
			return 0, 0, err
		}
		if found {
			return orderID, balance, nil
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

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
		if err == sql.ErrNoRows {
			return 0, 0, ErrProductNotBuyable
		}
		return 0, 0, err
	}
	if productStatus != productdomain.StatusOnSale && productStatus != productdomain.StatusPreSale {
		return 0, 0, ErrProductNotBuyable
	}
	if params.Quantity <= 0 || unitPriceCent > orderdomain.MaxOrderAmountCent/int64(params.Quantity) {
		return 0, 0, ErrOrderAmountTooLarge
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
		if isDuplicateKeyError(err) {
			_ = tx.Rollback()
			orderID, balance, found, replayErr := r.findExistingOrderPayment(ctx, params)
			if replayErr != nil {
				return 0, 0, replayErr
			}
			if found {
				return orderID, balance, nil
			}
			return 0, 0, ErrIdempotencyConflict
		}
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
	if len(statuses) == 0 {
		return nil, ErrOrderStatusInvalid
	}

	var query strings.Builder
	query.WriteString(`
SELECT id, product_id, product_name_snapshot, quantity, total_amount_cent, refund_amount_cent, status, created_at, shipped_at
FROM orders
WHERE buyer_id = ? AND status IN (`)
	args := []any{buyerID}
	for i, status := range statuses {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("?")
		args = append(args, status)
	}
	query.WriteString(`)
ORDER BY created_at DESC
LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)
	return r.queryBuyerOrders(ctx, query.String(), args...)
}

func (r *OrderRepository) queryBuyerOrders(ctx context.Context, query string, args ...any) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *OrderRepository) findExistingOrderPayment(ctx context.Context, params CreateOrderParams) (int64, int64, bool, error) {
	const query = `
SELECT p.user_id, p.order_id, p.type, p.status, o.product_id, o.quantity
FROM payment_transactions p
LEFT JOIN orders o ON o.id = p.order_id
WHERE p.idempotency_key = ?
LIMIT 1`
	var userID int64
	var orderID sql.NullInt64
	var paymentType int
	var paymentStatus int
	var productID sql.NullInt64
	var quantity sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, params.IdempotencyKey).Scan(&userID, &orderID, &paymentType, &paymentStatus, &productID, &quantity)
	if err == sql.ErrNoRows {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	if userID != params.BuyerID ||
		paymentType != 2 ||
		paymentStatus != 2 ||
		!orderID.Valid ||
		!productID.Valid ||
		productID.Int64 != params.ProductID ||
		!quantity.Valid ||
		int(quantity.Int64) != params.Quantity {
		return 0, 0, false, ErrIdempotencyConflict
	}
	balance, err := r.buyerBalance(ctx, params.BuyerID)
	if err != nil {
		return 0, 0, false, err
	}
	return orderID.Int64, balance, true, nil
}

func (r *OrderRepository) findExistingRefundPayment(ctx context.Context, buyerID, orderID int64, idempotencyKey string) (int64, bool, error) {
	const query = `
SELECT user_id, order_id, type, status
FROM payment_transactions
WHERE idempotency_key = ?
LIMIT 1`
	var userID int64
	var existingOrderID sql.NullInt64
	var paymentType int
	var paymentStatus int
	err := r.db.QueryRowContext(ctx, query, idempotencyKey).Scan(&userID, &existingOrderID, &paymentType, &paymentStatus)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if userID != buyerID ||
		paymentType != 3 ||
		paymentStatus != 2 ||
		!existingOrderID.Valid ||
		existingOrderID.Int64 != orderID {
		return 0, false, ErrIdempotencyConflict
	}
	balance, err := r.buyerBalance(ctx, buyerID)
	if err != nil {
		return 0, false, err
	}
	return balance, true, nil
}

func (r *OrderRepository) buyerBalance(ctx context.Context, buyerID int64) (int64, error) {
	var balance int64
	err := r.db.QueryRowContext(ctx, `SELECT balance_cent FROM buyer_profiles WHERE user_id = ?`, buyerID).Scan(&balance)
	return balance, err
}

func (r *OrderRepository) RefundOrder(ctx context.Context, buyerID, orderID int64, idempotencyKey string) (int64, error) {
	if idempotencyKey != "" {
		balance, found, err := r.findExistingRefundPayment(ctx, buyerID, orderID, idempotencyKey)
		if err != nil {
			return 0, err
		}
		if found {
			return balance, nil
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var sellerID int64
	var productID int64
	var quantity int
	var amountCent int64
	var status int
	const orderQuery = `
SELECT seller_id, product_id, quantity, total_amount_cent, status
FROM orders
WHERE id = ? AND buyer_id = ?
FOR UPDATE`
	if err := tx.QueryRowContext(ctx, orderQuery, orderID, buyerID).Scan(&sellerID, &productID, &quantity, &amountCent, &status); err != nil {
		return 0, err
	}
	if status != orderdomain.StatusPlacedUnshipped {
		return 0, ErrOrderStatusInvalid
	}

	const insertRefund = `
INSERT INTO payment_transactions (user_id, order_id, type, amount_cent, status, idempotency_key)
VALUES (?, ?, 3, ?, 2, ?)`
	if _, err := tx.ExecContext(ctx, insertRefund, buyerID, orderID, amountCent, idempotencyKey); err != nil {
		if isDuplicateKeyError(err) {
			_ = tx.Rollback()
			balance, found, replayErr := r.findExistingRefundPayment(ctx, buyerID, orderID, idempotencyKey)
			if replayErr != nil {
				return 0, replayErr
			}
			if found {
				return balance, nil
			}
			return 0, ErrIdempotencyConflict
		}
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE buyer_profiles SET balance_cent = balance_cent + ? WHERE user_id = ?`, amountCent, buyerID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE seller_profiles SET total_deal_amount_cent = total_deal_amount_cent - ? WHERE user_id = ?`, amountCent, sellerID); err != nil {
		return 0, err
	}

	var reserved int
	if err := tx.QueryRowContext(ctx, `SELECT reserved_quantity FROM product_inventory WHERE product_id = ? FOR UPDATE`, productID).Scan(&reserved); err != nil {
		return 0, err
	}
	if reserved >= quantity {
		const restoreInventory = `
UPDATE product_inventory
SET available_quantity = available_quantity + ?, reserved_quantity = reserved_quantity - ?
WHERE product_id = ?`
		if _, err := tx.ExecContext(ctx, restoreInventory, quantity, quantity, productID); err != nil {
			return 0, err
		}
	}

	const updateOrder = `
UPDATE orders
SET status = ?, refund_amount_cent = total_amount_cent, refunded_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateOrder, orderdomain.StatusRefunded, orderID); err != nil {
		return 0, err
	}
	if err := completeDelistingProduct(ctx, tx, sellerID, productID); err != nil {
		return 0, err
	}

	bizDate := time.Now().Format("2006-01-02")
	const upsertProductStats = `
INSERT INTO product_daily_stats (biz_date, product_id, seller_id, refund_amount_cent, refund_order_count)
VALUES (?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE refund_amount_cent = refund_amount_cent + VALUES(refund_amount_cent), refund_order_count = refund_order_count + 1`
	if _, err := tx.ExecContext(ctx, upsertProductStats, bizDate, productID, sellerID, amountCent); err != nil {
		return 0, err
	}
	const upsertSellerStats = `
INSERT INTO seller_daily_stats (biz_date, seller_id, refund_amount_cent, refund_order_count)
VALUES (?, ?, ?, 1)
ON DUPLICATE KEY UPDATE refund_amount_cent = refund_amount_cent + VALUES(refund_amount_cent), refund_order_count = refund_order_count + 1`
	if _, err := tx.ExecContext(ctx, upsertSellerStats, bizDate, sellerID, amountCent); err != nil {
		return 0, err
	}

	var balance int64
	if err := tx.QueryRowContext(ctx, `SELECT balance_cent FROM buyer_profiles WHERE user_id = ?`, buyerID).Scan(&balance); err != nil {
		return 0, err
	}
	return balance, tx.Commit()
}

func (r *OrderRepository) ReceiveOrder(ctx context.Context, buyerID, orderID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sellerID int64
	var productID int64
	var status int
	const selectOrder = `
SELECT seller_id, product_id, status
FROM orders
WHERE id = ? AND buyer_id = ?
FOR UPDATE`
	if err := tx.QueryRowContext(ctx, selectOrder, orderID, buyerID).Scan(&sellerID, &productID, &status); err != nil {
		return err
	}
	if status != orderdomain.StatusShipping {
		return ErrOrderStatusInvalid
	}
	const updateOrder = `
UPDATE orders
SET status = ?, is_deal_completed = 1, received_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateOrder, orderdomain.StatusReceived, orderID); err != nil {
		return err
	}
	if err := completeDelistingProduct(ctx, tx, sellerID, productID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *OrderRepository) ShipProductOrders(ctx context.Context, sellerID, productID int64) (int, int, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	var exists int
	const productQuery = `SELECT 1 FROM products WHERE id = ? AND seller_id = ? AND is_deleted = 0`
	if err := tx.QueryRowContext(ctx, productQuery, productID, sellerID).Scan(&exists); err != nil {
		return 0, 0, 0, err
	}

	const orderQuery = `
SELECT quantity
FROM orders
WHERE seller_id = ? AND product_id = ? AND status = ?
FOR UPDATE`
	rows, err := tx.QueryContext(ctx, orderQuery, sellerID, productID, orderdomain.StatusPlacedUnshipped)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	orderCount := 0
	shipQuantity := 0
	for rows.Next() {
		var quantity int
		if err := rows.Scan(&quantity); err != nil {
			return 0, 0, 0, err
		}
		orderCount++
		shipQuantity += quantity
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}
	if orderCount == 0 {
		return 0, 0, 0, tx.Commit()
	}

	var available int
	var reserved int
	if err := tx.QueryRowContext(ctx, `SELECT available_quantity, reserved_quantity FROM product_inventory WHERE product_id = ? FOR UPDATE`, productID).Scan(&available, &reserved); err != nil {
		return 0, 0, 0, err
	}
	useReserved := shipQuantity
	if reserved < useReserved {
		useReserved = reserved
	}
	needAvailable := shipQuantity - useReserved
	if available < needAvailable {
		return 0, 0, available, ErrInventoryNotEnough
	}

	const updateInventory = `
UPDATE product_inventory
SET available_quantity = available_quantity - ?,
    reserved_quantity = reserved_quantity - ?,
    shipped_quantity = shipped_quantity + ?
WHERE product_id = ?`
	if _, err := tx.ExecContext(ctx, updateInventory, needAvailable, useReserved, shipQuantity, productID); err != nil {
		return 0, 0, 0, err
	}
	const updateOrders = `
UPDATE orders
SET status = ?, shipped_at = CURRENT_TIMESTAMP(3)
WHERE seller_id = ? AND product_id = ? AND status = ?`
	if _, err := tx.ExecContext(ctx, updateOrders, orderdomain.StatusShipping, sellerID, productID, orderdomain.StatusPlacedUnshipped); err != nil {
		return 0, 0, 0, err
	}
	return orderCount, shipQuantity, available - needAvailable, tx.Commit()
}

func completeDelistingProduct(ctx context.Context, tx *sql.Tx, sellerID, productID int64) error {
	var productStatus int
	const selectProduct = `
SELECT status
FROM products
WHERE id = ? AND seller_id = ? AND is_deleted = 0
FOR UPDATE`
	if err := tx.QueryRowContext(ctx, selectProduct, productID, sellerID).Scan(&productStatus); err != nil {
		return err
	}
	if productStatus != productdomain.StatusDelisting {
		return nil
	}

	var unfinishedCount int
	const countOrders = `
SELECT COUNT(*)
FROM orders
WHERE product_id = ? AND seller_id = ? AND status IN (?, ?)`
	if err := tx.QueryRowContext(ctx, countOrders, productID, sellerID, orderdomain.StatusPlacedUnshipped, orderdomain.StatusShipping).Scan(&unfinishedCount); err != nil {
		return err
	}
	if unfinishedCount > 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE products SET status = ? WHERE id = ? AND seller_id = ?`, productdomain.StatusOffShelf, productID, sellerID)
	return err
}
