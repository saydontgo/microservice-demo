package auth

const (
	RoleBuyer  = "BUYER"
	RoleSeller = "SELLER"
)

const (
	UserStatusActive = 1
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	Status       int
}
