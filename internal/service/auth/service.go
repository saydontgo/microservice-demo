package authsvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"microservice-demo/internal/domain/auth"
	"microservice-demo/internal/repository"
)

var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrUserExists        = errors.New("user exists")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrRoleMismatch      = errors.New("role mismatch")
	ErrUserDisabled      = errors.New("user disabled")
)

type Repository interface {
	FindUserByUsername(ctx context.Context, username string) (auth.User, error)
	CreateBuyer(ctx context.Context, params repository.CreateBuyerParams) (int64, error)
	CreateSeller(ctx context.Context, params repository.CreateSellerParams) (int64, error)
}

type Service struct {
	repo        Repository
	passwordKey string
	tokenKey    string
	tokenTTL    time.Duration
	now         func() time.Time
}

type RegisterInput struct {
	Username        string
	Password        string
	Role            string
	Nickname        string
	AvatarURL       string
	Phone           string
	ShippingAddress string
	RegistrantName  string
	ShopName        string
	ShopAvatarURL   string
}

type RegisterOutput struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
}

type LoginInput struct {
	Username string
	Password string
	Role     string
}

type LoginOutput struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expiresIn"`
	Role      string `json:"role"`
}

type tokenPayload struct {
	UserID    int64  `json:"userId"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expiresAt"`
}

func NewService(repo Repository, passwordKey, tokenKey string, tokenTTL time.Duration) *Service {
	return &Service{repo: repo, passwordKey: passwordKey, tokenKey: tokenKey, tokenTTL: tokenTTL, now: time.Now}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Role = strings.TrimSpace(input.Role)
	if input.Username == "" || input.Password == "" || !validRole(input.Role) {
		return RegisterOutput{}, ErrInvalidArgument
	}
	if _, err := s.repo.FindUserByUsername(ctx, input.Username); err == nil {
		return RegisterOutput{}, ErrUserExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return RegisterOutput{}, err
	}

	passwordHash := s.hashPassword(input.Password)
	if input.Role == auth.RoleBuyer {
		userID, err := s.repo.CreateBuyer(ctx, repository.CreateBuyerParams{
			Username:        input.Username,
			PasswordHash:    passwordHash,
			Nickname:        valueOr(input.Nickname, input.Username),
			AvatarURL:       input.AvatarURL,
			Phone:           input.Phone,
			ShippingAddress: input.ShippingAddress,
		})
		return RegisterOutput{UserID: userID, Role: input.Role}, err
	}

	if strings.TrimSpace(input.RegistrantName) == "" || strings.TrimSpace(input.ShopName) == "" {
		return RegisterOutput{}, ErrInvalidArgument
	}
	userID, err := s.repo.CreateSeller(ctx, repository.CreateSellerParams{
		Username:       input.Username,
		PasswordHash:   passwordHash,
		RegistrantName: input.RegistrantName,
		ShopName:       input.ShopName,
		ShopAvatarURL:  input.ShopAvatarURL,
	})
	return RegisterOutput{UserID: userID, Role: input.Role}, err
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginOutput, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Role = strings.TrimSpace(input.Role)
	if input.Username == "" || input.Password == "" || !validRole(input.Role) {
		return LoginOutput{}, ErrInvalidArgument
	}

	user, err := s.repo.FindUserByUsername(ctx, input.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginOutput{}, ErrInvalidCredential
	}
	if err != nil {
		return LoginOutput{}, err
	}
	if user.Status != auth.UserStatusActive {
		return LoginOutput{}, ErrUserDisabled
	}
	if user.Role != input.Role {
		return LoginOutput{}, ErrRoleMismatch
	}
	if user.PasswordHash != s.hashPassword(input.Password) {
		return LoginOutput{}, ErrInvalidCredential
	}

	token, err := s.signToken(user)
	if err != nil {
		return LoginOutput{}, err
	}
	return LoginOutput{Token: token, ExpiresIn: int64(s.tokenTTL.Seconds()), Role: user.Role}, nil
}

func (s *Service) hashPassword(password string) string {
	sum := sha256.Sum256([]byte(s.passwordKey + ":" + password))
	return hex.EncodeToString(sum[:])
}

func (s *Service) signToken(user auth.User) (string, error) {
	payload := tokenPayload{UserID: user.ID, Role: user.Role, ExpiresAt: s.now().Add(s.tokenTTL).Unix()}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadText := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, []byte(s.tokenKey))
	_, _ = mac.Write([]byte(payloadText))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s", payloadText, signature), nil
}

func validRole(role string) bool {
	return role == auth.RoleBuyer || role == auth.RoleSeller
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
