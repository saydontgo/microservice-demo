package repository

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrBalanceNotEnough     = errors.New("balance not enough")
	ErrInventoryNotEnough   = errors.New("inventory not enough")
	ErrOrderStatusInvalid   = errors.New("order status invalid")
	ErrOrderAmountTooLarge  = errors.New("order amount too large")
	ErrProductNotBuyable    = errors.New("product not buyable")
	ErrProductStatusInvalid = errors.New("product status invalid")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
)

func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
