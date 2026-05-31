package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type ProductRepository struct {
	db *sql.DB
}

type Product struct {
	ID                int64
	SellerID          int64
	ProductName       string
	Description       sql.NullString
	PriceCent         int64
	Status            int
	AvailableQuantity int
	ShopName          sql.NullString
}

type CreateProductParams struct {
	SellerID         int64
	ProductName      string
	Description      string
	PriceCent        int64
	Status           int
	InitialInventory int
}

type UpdateProductParams struct {
	ProductID   int64
	SellerID    int64
	ProductName string
	Description string
	PriceCent   int64
	Status      int
}

type SellerProductFilter struct {
	SellerID          int64
	StartDate         string
	EndDate           string
	Status            *int
	ProductID         *int64
	ProductNamePrefix string
	Shipped           *bool
	Limit             int
	Offset            int
}

type SellerProduct struct {
	Product
	DealAmountCent   int64
	RefundAmountCent int64
	RefundRate       float64
}

type TrendPoint struct {
	Date             string
	DealAmountCent   int64
	RefundAmountCent int64
	RefundRate       float64
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) CreateProduct(ctx context.Context, params CreateProductParams) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	const insertProduct = `
INSERT INTO products (seller_id, product_name, description, price_cent, status, is_deleted)
VALUES (?, ?, ?, ?, ?, 0)`
	result, err := tx.ExecContext(ctx, insertProduct, params.SellerID, params.ProductName, nullable(params.Description), params.PriceCent, params.Status)
	if err != nil {
		return 0, err
	}
	productID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	const insertInventory = `
INSERT INTO product_inventory (product_id, available_quantity, reserved_quantity, shipped_quantity)
VALUES (?, ?, 0, 0)`
	if _, err := tx.ExecContext(ctx, insertInventory, productID, params.InitialInventory); err != nil {
		return 0, err
	}
	return productID, tx.Commit()
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, params UpdateProductParams) error {
	const query = `
UPDATE products
SET product_name = ?, description = ?, price_cent = ?, status = ?
WHERE id = ? AND seller_id = ? AND is_deleted = 0`
	result, err := r.db.ExecContext(ctx, query, params.ProductName, nullable(params.Description), params.PriceCent, params.Status, params.ProductID, params.SellerID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ProductRepository) AddInventory(ctx context.Context, sellerID, productID int64, quantity int) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var exists int
	const checkProduct = `SELECT 1 FROM products WHERE id = ? AND seller_id = ? AND is_deleted = 0`
	if err := tx.QueryRowContext(ctx, checkProduct, productID, sellerID).Scan(&exists); err != nil {
		return 0, err
	}

	const updateInventory = `
UPDATE product_inventory
SET available_quantity = available_quantity + ?
WHERE product_id = ?`
	if _, err := tx.ExecContext(ctx, updateInventory, quantity, productID); err != nil {
		return 0, err
	}

	var available int
	if err := tx.QueryRowContext(ctx, `SELECT available_quantity FROM product_inventory WHERE product_id = ?`, productID).Scan(&available); err != nil {
		return 0, err
	}
	return available, tx.Commit()
}

func (r *ProductRepository) SearchBuyerProducts(ctx context.Context, namePrefix string, limit, offset int) ([]Product, error) {
	const query = `
SELECT p.id, p.seller_id, p.product_name, p.description, p.price_cent, p.status,
       CASE WHEN p.status = 2 THEN 0 ELSE i.available_quantity END AS display_inventory,
       sp.shop_name
FROM products p
JOIN product_inventory i ON i.product_id = p.id
JOIN seller_profiles sp ON sp.user_id = p.seller_id
WHERE p.product_name LIKE CONCAT(?, '%')
  AND p.status IN (1, 2)
  AND p.is_deleted = 0
ORDER BY p.created_at DESC
LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, namePrefix, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var item Product
		if err := rows.Scan(&item.ID, &item.SellerID, &item.ProductName, &item.Description, &item.PriceCent, &item.Status, &item.AvailableQuantity, &item.ShopName); err != nil {
			return nil, err
		}
		products = append(products, item)
	}
	return products, rows.Err()
}

func (r *ProductRepository) ListSellerProducts(ctx context.Context, filter SellerProductFilter) ([]SellerProduct, error) {
	var query strings.Builder
	query.WriteString(`
SELECT p.id, p.seller_id, p.product_name, p.description, p.price_cent, p.status,
       CASE WHEN p.status = 2 THEN 0 ELSE i.available_quantity END AS display_inventory,
       COALESCE(SUM(s.deal_amount_cent), 0) AS deal_amount_cent,
       COALESCE(SUM(s.refund_amount_cent), 0) AS refund_amount_cent
FROM products p
JOIN product_inventory i ON i.product_id = p.id
LEFT JOIN product_daily_stats s
  ON s.product_id = p.id
 AND s.biz_date BETWEEN ? AND ?
WHERE p.seller_id = ?
  AND p.is_deleted = 0`)
	args := []any{filter.StartDate, filter.EndDate, filter.SellerID}
	if filter.Status != nil {
		query.WriteString(" AND p.status = ?")
		args = append(args, *filter.Status)
	}
	if filter.ProductID != nil {
		query.WriteString(" AND p.id = ?")
		args = append(args, *filter.ProductID)
	}
	if filter.ProductNamePrefix != "" {
		query.WriteString(" AND p.product_name LIKE CONCAT(?, '%')")
		args = append(args, filter.ProductNamePrefix)
	}
	if filter.Shipped != nil && *filter.Shipped {
		query.WriteString(" AND EXISTS (SELECT 1 FROM orders o WHERE o.product_id = p.id AND o.seller_id = p.seller_id AND o.status IN (2, 3))")
	}
	if filter.Shipped != nil && !*filter.Shipped {
		query.WriteString(" AND EXISTS (SELECT 1 FROM orders o WHERE o.product_id = p.id AND o.seller_id = p.seller_id AND o.status = 1)")
	}
	query.WriteString(`
GROUP BY p.id, p.seller_id, p.product_name, p.description, p.price_cent, p.status, i.available_quantity
ORDER BY p.created_at DESC
LIMIT ? OFFSET ?`)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]SellerProduct, 0)
	for rows.Next() {
		var item SellerProduct
		if err := rows.Scan(&item.ID, &item.SellerID, &item.ProductName, &item.Description, &item.PriceCent, &item.Status, &item.AvailableQuantity, &item.DealAmountCent, &item.RefundAmountCent); err != nil {
			return nil, err
		}
		if item.DealAmountCent > 0 {
			item.RefundRate = float64(item.RefundAmountCent) / float64(item.DealAmountCent)
		}
		products = append(products, item)
	}
	return products, rows.Err()
}

func (r *ProductRepository) ListSellerTrend(ctx context.Context, sellerID int64, startDate, endDate string) ([]TrendPoint, error) {
	const query = `
SELECT biz_date, deal_amount_cent, refund_amount_cent
FROM seller_daily_stats
WHERE seller_id = ?
  AND biz_date BETWEEN ? AND ?
ORDER BY biz_date ASC`
	rows, err := r.db.QueryContext(ctx, query, sellerID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]TrendPoint, 0)
	for rows.Next() {
		var day time.Time
		var point TrendPoint
		if err := rows.Scan(&day, &point.DealAmountCent, &point.RefundAmountCent); err != nil {
			return nil, err
		}
		point.Date = day.Format("2006-01-02")
		if point.DealAmountCent > 0 {
			point.RefundRate = float64(point.RefundAmountCent) / float64(point.DealAmountCent)
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (r *ProductRepository) DelistProduct(ctx context.Context, sellerID, productID int64) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var status int
	const productQuery = `
SELECT status
FROM products
WHERE id = ? AND seller_id = ? AND status = 1 AND is_deleted = 0
FOR UPDATE`
	if err := tx.QueryRowContext(ctx, productQuery, productID, sellerID).Scan(&status); err != nil {
		return 0, err
	}

	nextStatus := 3
	var unfinishedCount int
	const orderQuery = `
SELECT COUNT(*)
FROM orders
WHERE product_id = ? AND seller_id = ? AND status IN (1, 2)`
	if err := tx.QueryRowContext(ctx, orderQuery, productID, sellerID).Scan(&unfinishedCount); err != nil {
		return 0, err
	}
	if unfinishedCount == 0 {
		nextStatus = 4
	}

	const updateQuery = `
UPDATE products
SET status = ?
WHERE id = ? AND seller_id = ?`
	if _, err := tx.ExecContext(ctx, updateQuery, nextStatus, productID, sellerID); err != nil {
		return 0, err
	}
	return nextStatus, tx.Commit()
}
