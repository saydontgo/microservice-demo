package repository

import (
	"context"
	"database/sql"
)

type UserRepository struct {
	db *sql.DB
}

type BuyerProfile struct {
	UserID          int64
	Nickname        string
	AvatarURL       sql.NullString
	Phone           sql.NullString
	ShippingAddress sql.NullString
	BalanceCent     int64
}

type SellerProfile struct {
	UserID              int64
	RegistrantName      string
	ShopName            string
	ShopAvatarURL       sql.NullString
	Theme               string
	TotalDealAmountCent int64
}

type UpdateBuyerProfileParams struct {
	UserID          int64
	Nickname        string
	AvatarURL       string
	Phone           string
	ShippingAddress string
}

type UpdateSellerProfileParams struct {
	UserID         int64
	RegistrantName string
	ShopName       string
	ShopAvatarURL  string
	Theme          string
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetBuyerProfile(ctx context.Context, userID int64) (BuyerProfile, error) {
	const query = `
SELECT user_id, nickname, avatar_url, phone, shipping_address, balance_cent
FROM buyer_profiles
WHERE user_id = ?`
	var profile BuyerProfile
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&profile.UserID, &profile.Nickname, &profile.AvatarURL, &profile.Phone, &profile.ShippingAddress, &profile.BalanceCent)
	return profile, err
}

func (r *UserRepository) UpdateBuyerProfile(ctx context.Context, params UpdateBuyerProfileParams) error {
	const query = `
UPDATE buyer_profiles
SET nickname = ?, avatar_url = ?, phone = ?, shipping_address = ?
WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, params.Nickname, nullable(params.AvatarURL), nullable(params.Phone), nullable(params.ShippingAddress), params.UserID)
	return err
}

func (r *UserRepository) RechargeBuyerBalance(ctx context.Context, userID int64, amountCent int64, idempotencyKey string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	const insertTx = `
INSERT INTO payment_transactions (user_id, type, amount_cent, status, idempotency_key)
VALUES (?, 1, ?, 2, ?)`
	if _, err := tx.ExecContext(ctx, insertTx, userID, amountCent, idempotencyKey); err != nil {
		return 0, err
	}

	const updateBalance = `
UPDATE buyer_profiles
SET balance_cent = balance_cent + ?
WHERE user_id = ?`
	if _, err := tx.ExecContext(ctx, updateBalance, amountCent, userID); err != nil {
		return 0, err
	}

	var balance int64
	if err := tx.QueryRowContext(ctx, `SELECT balance_cent FROM buyer_profiles WHERE user_id = ?`, userID).Scan(&balance); err != nil {
		return 0, err
	}
	return balance, tx.Commit()
}

func (r *UserRepository) GetSellerProfile(ctx context.Context, userID int64) (SellerProfile, error) {
	const query = `
SELECT user_id, registrant_name, shop_name, shop_avatar_url, theme, total_deal_amount_cent
FROM seller_profiles
WHERE user_id = ?`
	var profile SellerProfile
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&profile.UserID, &profile.RegistrantName, &profile.ShopName, &profile.ShopAvatarURL, &profile.Theme, &profile.TotalDealAmountCent)
	return profile, err
}

func (r *UserRepository) UpdateSellerProfile(ctx context.Context, params UpdateSellerProfileParams) error {
	const query = `
UPDATE seller_profiles
SET registrant_name = ?, shop_name = ?, shop_avatar_url = ?, theme = ?
WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, params.RegistrantName, params.ShopName, nullable(params.ShopAvatarURL), params.Theme, params.UserID)
	return err
}
