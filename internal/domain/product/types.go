package product

const (
	StatusOnSale    = 1
	StatusPreSale   = 2
	StatusDelisting = 3
	StatusOffShelf  = 4
)

func ValidStatus(status int) bool {
	return status == StatusOnSale || status == StatusPreSale || status == StatusDelisting || status == StatusOffShelf
}

func StatusName(status int) string {
	switch status {
	case StatusOnSale:
		return "ON_SALE"
	case StatusPreSale:
		return "PRE_SALE"
	case StatusDelisting:
		return "DELISTING"
	case StatusOffShelf:
		return "OFF_SHELF"
	default:
		return "UNKNOWN"
	}
}
