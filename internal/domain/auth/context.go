package auth

import "context"

type CurrentUser struct {
	ID   int64
	Role string
}

type contextKey struct{}

func WithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

func CurrentUserFromContext(ctx context.Context) (CurrentUser, bool) {
	user, ok := ctx.Value(contextKey{}).(CurrentUser)
	return user, ok
}
