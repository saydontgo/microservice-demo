package repository

import (
	"context"
	"database/sql"
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
