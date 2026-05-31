package productsvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

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
	ListSellerProducts(ctx context.Context, filter repository.SellerProductFilter) ([]repository.SellerProduct, error)
	ListSellerTrend(ctx context.Context, sellerID int64, startDate, endDate string) ([]repository.TrendPoint, error)
	DelistProduct(ctx context.Context, sellerID, productID int64) error
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

type SellerProductListInput struct {
	StartDate         string
	EndDate           string
	Status            *int
	ProductID         *int64
	ProductNamePrefix string
	Shipped           *bool
	Page              int
	PageSize          int
}

type SellerProductOutput struct {
	ProductOutput
	DealAmountCent   int64   `json:"dealAmountCent"`
	RefundAmountCent int64   `json:"refundAmountCent"`
	RefundRate       float64 `json:"refundRate"`
}

type TrendInput struct {
	StartDate string
	EndDate   string
	Days      int
}

type TrendOutput struct {
	Points []TrendPointOutput `json:"points"`
}

type TrendPointOutput struct {
	Date             string  `json:"date"`
	DealAmountCent   int64   `json:"dealAmountCent"`
	RefundAmountCent int64   `json:"refundAmountCent"`
	RefundRate       float64 `json:"refundRate"`
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

func (s *Service) ListSellerProducts(ctx context.Context, input SellerProductListInput) ([]SellerProductOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return nil, err
	}
	startDate, endDate, err := dateRange(input.StartDate, input.EndDate, 7)
	if err != nil || input.Page <= 0 || input.PageSize <= 0 || input.PageSize > 100 {
		return nil, ErrInvalidArgument
	}
	if input.Status != nil && !productdomain.ValidStatus(*input.Status) {
		return nil, ErrInvalidArgument
	}
	items, err := s.repo.ListSellerProducts(ctx, repository.SellerProductFilter{
		SellerID:          user.ID,
		StartDate:         startDate,
		EndDate:           endDate,
		Status:            input.Status,
		ProductID:         input.ProductID,
		ProductNamePrefix: strings.TrimSpace(input.ProductNamePrefix),
		Shipped:           input.Shipped,
		Limit:             input.PageSize,
		Offset:            (input.Page - 1) * input.PageSize,
	})
	if err != nil {
		return nil, err
	}
	outputs := make([]SellerProductOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, SellerProductOutput{
			ProductOutput: ProductOutput{
				ProductID:        item.ID,
				ProductName:      item.ProductName,
				Description:      nullString(item.Description),
				PriceCent:        item.PriceCent,
				Status:           item.Status,
				StatusName:       productdomain.StatusName(item.Status),
				DisplayInventory: item.AvailableQuantity,
			},
			DealAmountCent:   item.DealAmountCent,
			RefundAmountCent: item.RefundAmountCent,
			RefundRate:       item.RefundRate,
		})
	}
	return outputs, nil
}

func (s *Service) ListSellerTrend(ctx context.Context, input TrendInput) (TrendOutput, error) {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return TrendOutput{}, err
	}
	days := input.Days
	if days <= 0 {
		days = 7
	}
	startDate, endDate, err := dateRange(input.StartDate, input.EndDate, days)
	if err != nil {
		return TrendOutput{}, ErrInvalidArgument
	}
	points, err := s.repo.ListSellerTrend(ctx, user.ID, startDate, endDate)
	if err != nil {
		return TrendOutput{}, err
	}
	return TrendOutput{Points: fillTrendPoints(startDate, endDate, points)}, nil
}

func (s *Service) DelistProduct(ctx context.Context, productID int64) error {
	user, err := currentUser(ctx, auth.RoleSeller)
	if err != nil {
		return err
	}
	if productID <= 0 {
		return ErrInvalidArgument
	}
	err = s.repo.DelistProduct(ctx, user.ID, productID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrProductNotFound
	}
	return err
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

func dateRange(startDate string, endDate string, defaultDays int) (string, string, error) {
	if startDate == "" && endDate == "" {
		end := time.Now()
		start := end.AddDate(0, 0, -defaultDays+1)
		return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", "", err
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil || end.Before(start) {
		return "", "", ErrInvalidArgument
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

func fillTrendPoints(startDate string, endDate string, points []repository.TrendPoint) []TrendPointOutput {
	byDate := make(map[string]repository.TrendPoint, len(points))
	for _, point := range points {
		byDate[point.Date] = point
	}
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	outputs := make([]TrendPointOutput, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		point := byDate[date]
		outputs = append(outputs, TrendPointOutput{
			Date:             date,
			DealAmountCent:   point.DealAmountCent,
			RefundAmountCent: point.RefundAmountCent,
			RefundRate:       point.RefundRate,
		})
	}
	return outputs
}
