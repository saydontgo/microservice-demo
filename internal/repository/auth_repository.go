package repository

import (
	"context"
	"database/sql"

	"microservice-demo/internal/domain/auth"
)

type AuthRepository struct {
	db *sql.DB
}

type CreateBuyerParams struct {
	Username        string
	PasswordHash    string
	Nickname        string
	AvatarURL       string
	Phone           string
	ShippingAddress string
}

type CreateSellerParams struct {
	Username       string
	PasswordHash   string
	RegistrantName string
	ShopName       string
	ShopAvatarURL  string
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindUserByUsername(ctx context.Context, username string) (auth.User, error) {
	const query = `
SELECT id, username, password_hash, role, status
FROM users
WHERE username = ?
LIMIT 1`

	var user auth.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status)
	return user, err
}

func (r *AuthRepository) CreateBuyer(ctx context.Context, params CreateBuyerParams) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	userID, err := insertUser(ctx, tx, params.Username, params.PasswordHash, auth.RoleBuyer)
	if err != nil {
		return 0, err
	}

	const insertProfile = `
INSERT INTO buyer_profiles (user_id, nickname, avatar_url, phone, shipping_address)
VALUES (?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertProfile, userID, params.Nickname, nullable(params.AvatarURL), nullable(params.Phone), nullable(params.ShippingAddress)); err != nil {
		return 0, err
	}
	return userID, tx.Commit()
}

func (r *AuthRepository) CreateSeller(ctx context.Context, params CreateSellerParams) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	userID, err := insertUser(ctx, tx, params.Username, params.PasswordHash, auth.RoleSeller)
	if err != nil {
		return 0, err
	}

	const insertProfile = `
INSERT INTO seller_profiles (user_id, registrant_name, shop_name, shop_avatar_url)
VALUES (?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertProfile, userID, params.RegistrantName, params.ShopName, nullable(params.ShopAvatarURL)); err != nil {
		return 0, err
	}
	return userID, tx.Commit()
}

func insertUser(ctx context.Context, tx *sql.Tx, username, passwordHash, role string) (int64, error) {
	const query = `
INSERT INTO users (username, password_hash, role, status)
VALUES (?, ?, ?, ?)`
	result, err := tx.ExecContext(ctx, query, username, passwordHash, role, auth.UserStatusActive)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
