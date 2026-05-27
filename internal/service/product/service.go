package productsvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"microservice-demo/internal/domain/auth"
	productdomain "microservice-demo/internal/domain/product"
	"microservice-demo/internal/repository"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrProductNotFound = errors.New("product not found")
)

type Repository interface {
	CreateProduct(ctx context.Context, params repository.CreateProductParams) (int64, error)
	UpdateProduct(ctx context.Context, params repository.UpdateProductParams) error
	AddInventory(ctx context.Context, sellerID, productID int64, quantity int) (int, error)
	SearchBuyerProducts(ctx context.Context, namePrefix string, limit, offset int) ([]repository.Product, error)
}

type Service struct {
	repo Repository
}

type CreateProductInput struct {
	ProductName      string
	Description      string
	PriceCent        int64
	Status           int
	InitialInventory int
}

type UpdateProductInput struct {
	ProductID   int64
	ProductName string
	Description string
	PriceCent   int64
	Status      int
}

type ProductOutput struct {
	ProductID        int64  `json:"productId"`
	ProductName      string `json:"productName"`
	Description      string `json:"description"`
	PriceCent        int64  `json:"priceCent"`
	Status           int    `json:"status"`
	StatusName       string `json:"statusName"`
	DisplayInventory int    `json:"displayInventory"`
	ShopName         string `json:"shopName,omitempty"`
}

type CreateProductOutput struct {
	ProductID int64 `json:"productId"`
	Status    int   `json:"status"`
}

type InventoryOutput struct {
	ProductID         int64 `json:"productId"`
	AvailableQuantity int   `json:"availableQuantity"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateProduct(ctx context.Context, input CreateProductInput) (CreateProductOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return CreateProductOutput{}, err
	}
	if !validProductInput(input.ProductName, input.PriceCent, input.Status) || input.InitialInventory < 0 {
		return CreateProductOutput{}, ErrInvalidArgument
	}
	productID, err := s.repo.CreateProduct(ctx, repository.CreateProductParams{
		SellerID:         user.ID,
		ProductName:      strings.TrimSpace(input.ProductName),
		Description:      input.Description,
		PriceCent:        input.PriceCent,
		Status:           input.Status,
		InitialInventory: input.InitialInventory,
	})
	return CreateProductOutput{ProductID: productID, Status: input.Status}, err
}

func (s *Service) UpdateProduct(ctx context.Context, input UpdateProductInput) error {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return err
	}
	if input.ProductID <= 0 || !validProductInput(input.ProductName, input.PriceCent, input.Status) {
		return ErrInvalidArgument
	}
	err = s.repo.UpdateProduct(ctx, repository.UpdateProductParams{
		ProductID:   input.ProductID,
		SellerID:    user.ID,
		ProductName: strings.TrimSpace(input.ProductName),
		Description: input.Description,
		PriceCent:   input.PriceCent,
		Status:      input.Status,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrProductNotFound
	}
	return err
}

func (s *Service) AddInventory(ctx context.Context, productID int64, quantity int) (InventoryOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return InventoryOutput{}, err
	}
	if productID <= 0 || quantity <= 0 {
		return InventoryOutput{}, ErrInvalidArgument
	}
	available, err := s.repo.AddInventory(ctx, user.ID, productID, quantity)
	if errors.Is(err, sql.ErrNoRows) {
		return InventoryOutput{}, ErrProductNotFound
	}
	return InventoryOutput{ProductID: productID, AvailableQuantity: available}, err
}

func (s *Service) SearchBuyerProducts(ctx context.Context, namePrefix string, page, pageSize int) ([]ProductOutput, error) {
	if _, err := currentUser(ctx, auth.RoleBuyer); err != nil {
		return nil, err
	}
	if strings.TrimSpace(namePrefix) == "" || page <= 0 || pageSize <= 0 || pageSize > 100 {
		return nil, ErrInvalidArgument
	}
	items, err := s.repo.SearchBuyerProducts(ctx, strings.TrimSpace(namePrefix), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	outputs := make([]ProductOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, ProductOutput{
			ProductID:        item.ID,
			ProductName:      item.ProductName,
			Description:      nullString(item.Description),
			PriceCent:        item.PriceCent,
			Status:           item.Status,
			StatusName:       productdomain.StatusName(item.Status),
			DisplayInventory: item.AvailableQuantity,
			ShopName:         nullString(item.ShopName),
		})
	}
	return outputs, nil
}

func currentUser(ctx context.Context, role string) (auth.CurrentUser, error) {
	user, ok := auth.CurrentUserFromContext(ctx)
	if !ok || user.Role != role {
		return auth.CurrentUser{}, ErrUnauthenticated
	}
	return user, nil
}

func validProductInput(name string, priceCent int64, status int) bool {
	return strings.TrimSpace(name) != "" && priceCent > 0 && productdomain.ValidStatus(status)
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}
