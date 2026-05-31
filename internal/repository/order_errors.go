package repository

import "errors"

var (
	ErrBalanceNotEnough   = errors.New("balance not enough")
	ErrInventoryNotEnough = errors.New("inventory not enough")
)
