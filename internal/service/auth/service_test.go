package authsvc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"microservice-demo/internal/domain/auth"
	"microservice-demo/internal/repository"
)

type fakeRepo struct {
	user     auth.User
	findErr  error
	buyerID  int64
	sellerID int64
	buyerIn  repository.CreateBuyerParams
	sellerIn repository.CreateSellerParams
}

func (r *fakeRepo) FindUserByUsername(context.Context, string) (auth.User, error) {
	return r.user, r.findErr
}

func (r *fakeRepo) CreateBuyer(_ context.Context, params repository.CreateBuyerParams) (int64, error) {
	r.buyerIn = params
	return r.buyerID, nil
}

func (r *fakeRepo) CreateSeller(_ context.Context, params repository.CreateSellerParams) (int64, error) {
	r.sellerIn = params
	return r.sellerID, nil
}

func TestServiceRegister_BitsUT(t *testing.T) {
	t.Run("注册买家成功", func(t *testing.T) {
		repo := &fakeRepo{findErr: sql.ErrNoRows, buyerID: 1001}
		svc := NewService(repo, "salt", "secret", time.Hour)

		got, err := svc.Register(context.Background(), RegisterInput{Username: "buyer1", Password: "pwd", Role: auth.RoleBuyer})

		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if got.UserID != 1001 || got.Role != auth.RoleBuyer {
			t.Fatalf("Register() = %+v", got)
		}
		if repo.buyerIn.Nickname != "buyer1" {
			t.Fatalf("buyer nickname = %q, want buyer1", repo.buyerIn.Nickname)
		}
	})

	t.Run("用户名已存在", func(t *testing.T) {
		repo := &fakeRepo{user: auth.User{ID: 1}}
		svc := NewService(repo, "salt", "secret", time.Hour)

		_, err := svc.Register(context.Background(), RegisterInput{Username: "buyer1", Password: "pwd", Role: auth.RoleBuyer})

		if err != ErrUserExists {
			t.Fatalf("Register() error = %v, want ErrUserExists", err)
		}
	})
}

func TestServiceLogin_BitsUT(t *testing.T) {
	t.Run("登录成功返回token", func(t *testing.T) {
		repo := &fakeRepo{findErr: sql.ErrNoRows}
		svc := NewService(repo, "salt", "secret", time.Hour)
		hash := svc.hashPassword("pwd")
		repo.findErr = nil
		repo.user = auth.User{ID: 1001, Username: "buyer1", PasswordHash: hash, Role: auth.RoleBuyer, Status: auth.UserStatusActive}

		got, err := svc.Login(context.Background(), LoginInput{Username: "buyer1", Password: "pwd", Role: auth.RoleBuyer})

		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if got.Token == "" || got.ExpiresIn != 3600 || got.Role != auth.RoleBuyer {
			t.Fatalf("Login() = %+v", got)
		}
	})

	t.Run("密码错误", func(t *testing.T) {
		repo := &fakeRepo{user: auth.User{ID: 1001, PasswordHash: "bad", Role: auth.RoleBuyer, Status: auth.UserStatusActive}}
		svc := NewService(repo, "salt", "secret", time.Hour)

		_, err := svc.Login(context.Background(), LoginInput{Username: "buyer1", Password: "pwd", Role: auth.RoleBuyer})

		if err != ErrInvalidCredential {
			t.Fatalf("Login() error = %v, want ErrInvalidCredential", err)
		}
	})

	t.Run("身份不匹配", func(t *testing.T) {
		repo := &fakeRepo{user: auth.User{ID: 1001, PasswordHash: "hash", Role: auth.RoleSeller, Status: auth.UserStatusActive}}
		svc := NewService(repo, "salt", "secret", time.Hour)

		_, err := svc.Login(context.Background(), LoginInput{Username: "seller1", Password: "pwd", Role: auth.RoleBuyer})

		if err != ErrRoleMismatch {
			t.Fatalf("Login() error = %v, want ErrRoleMismatch", err)
		}
	})
}
