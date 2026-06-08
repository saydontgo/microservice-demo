package usersvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"microservice-demo/internal/domain/auth"
	"microservice-demo/internal/repository"
)

var (
	ErrUnauthenticated     = errors.New("unauthenticated")
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrProfileNotFound     = errors.New("profile not found")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

type Repository interface {
	GetBuyerProfile(ctx context.Context, userID int64) (repository.BuyerProfile, error)
	UpdateBuyerProfile(ctx context.Context, params repository.UpdateBuyerProfileParams) error
	RechargeBuyerBalance(ctx context.Context, userID int64, amountCent int64, idempotencyKey string) (int64, error)
	GetSellerProfile(ctx context.Context, userID int64) (repository.SellerProfile, error)
	UpdateSellerProfile(ctx context.Context, params repository.UpdateSellerProfileParams) error
}

type Service struct {
	repo Repository
}

type BuyerProfileOutput struct {
	UserID          int64  `json:"userId"`
	Nickname        string `json:"nickname"`
	AvatarURL       string `json:"avatarUrl"`
	Phone           string `json:"phone"`
	ShippingAddress string `json:"shippingAddress"`
	BalanceCent     int64  `json:"balanceCent"`
}

type UpdateBuyerProfileInput struct {
	Nickname        string
	AvatarURL       string
	Phone           string
	ShippingAddress string
}

type RechargeOutput struct {
	BalanceCent int64 `json:"balanceCent"`
}

type SellerProfileOutput struct {
	UserID              int64  `json:"userId"`
	RegistrantName      string `json:"registrantName"`
	ShopName            string `json:"shopName"`
	ShopAvatarURL       string `json:"shopAvatarUrl"`
	Theme               string `json:"theme"`
	TotalDealAmountCent int64  `json:"totalDealAmountCent"`
}

type UpdateSellerProfileInput struct {
	RegistrantName string
	ShopName       string
	ShopAvatarURL  string
	Theme          string
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetBuyerProfile(ctx context.Context) (BuyerProfileOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return BuyerProfileOutput{}, err
	}
	profile, err := s.repo.GetBuyerProfile(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return BuyerProfileOutput{}, ErrProfileNotFound
	}
	if err != nil {
		return BuyerProfileOutput{}, err
	}
	return BuyerProfileOutput{
		UserID:          profile.UserID,
		Nickname:        profile.Nickname,
		AvatarURL:       nullString(profile.AvatarURL),
		Phone:           nullString(profile.Phone),
		ShippingAddress: nullString(profile.ShippingAddress),
		BalanceCent:     profile.BalanceCent,
	}, nil
}

func (s *Service) UpdateBuyerProfile(ctx context.Context, input UpdateBuyerProfileInput) error {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Nickname) == "" {
		return ErrInvalidArgument
	}
	return s.repo.UpdateBuyerProfile(ctx, repository.UpdateBuyerProfileParams{
		UserID:          user.ID,
		Nickname:        input.Nickname,
		AvatarURL:       input.AvatarURL,
		Phone:           input.Phone,
		ShippingAddress: input.ShippingAddress,
	})
}

func (s *Service) RechargeBuyerBalance(ctx context.Context, amountCent int64, idempotencyKey string) (RechargeOutput, error) {
	user, err := currentUser(ctx, auth.RoleBuyer)
	if err != nil {
		return RechargeOutput{}, err
	}
	if amountCent <= 0 || strings.TrimSpace(idempotencyKey) == "" {
		return RechargeOutput{}, ErrInvalidArgument
	}
	balance, err := s.repo.RechargeBuyerBalance(ctx, user.ID, amountCent, idempotencyKey)
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		return RechargeOutput{}, ErrIdempotencyConflict
	}
	return RechargeOutput{BalanceCent: balance}, err
}

func (s *Service) GetSellerProfile(ctx context.Context) (SellerProfileOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return SellerProfileOutput{}, err
	}
	profile, err := s.repo.GetSellerProfile(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return SellerProfileOutput{}, ErrProfileNotFound
	}
	if err != nil {
		return SellerProfileOutput{}, err
	}
	return SellerProfileOutput{
		UserID:              profile.UserID,
		RegistrantName:      profile.RegistrantName,
		ShopName:            profile.ShopName,
		ShopAvatarURL:       nullString(profile.ShopAvatarURL),
		Theme:               profile.Theme,
		TotalDealAmountCent: profile.TotalDealAmountCent,
	}, nil
}

func (s *Service) UpdateSellerProfile(ctx context.Context, input UpdateSellerProfileInput) error {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.RegistrantName) == "" || strings.TrimSpace(input.ShopName) == "" || !validTheme(input.Theme) {
		return ErrInvalidArgument
	}
	return s.repo.UpdateSellerProfile(ctx, repository.UpdateSellerProfileParams{
		UserID:         user.ID,
		RegistrantName: input.RegistrantName,
		ShopName:       input.ShopName,
		ShopAvatarURL:  input.ShopAvatarURL,
		Theme:          input.Theme,
	})
}

func currentUser(ctx context.Context, role string) (auth.CurrentUser, error) {
	user, ok := auth.CurrentUserFromContext(ctx)
	if !ok || user.Role != role {
		return auth.CurrentUser{}, ErrUnauthenticated
	}
	return user, nil
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func validTheme(theme string) bool {
	return theme == "LIGHT" || theme == "DARK"
}
