package order

const (
	StatusPlacedUnshipped = 1
	StatusShipping        = 2
	StatusReceived        = 3
	StatusRefunded        = 4
)

const (
	MaxBuyQuantity     = 100
	MaxOrderAmountCent = 10000000000
)

func StatusName(status int) string {
	switch status {
	case StatusPlacedUnshipped:
		return "PLACED_UNSHIPPED"
	case StatusShipping:
		return "SHIPPING"
	case StatusReceived:
		return "RECEIVED"
	case StatusRefunded:
		return "REFUNDED"
	default:
		return "UNKNOWN"
	}
}
