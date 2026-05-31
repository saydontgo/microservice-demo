package repository

import "errors"

var (
	ErrBalanceNotEnough   = errors.New("balance not enough")
	ErrInventoryNotEnough = errors.New("inventory not enough")
	ErrOrderStatusInvalid = errors.New("order status invalid")
)
