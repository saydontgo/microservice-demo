package middleware

import (
	"context"
	"net/http"
	"strings"

	"microservice-demo/internal/domain/auth"
	"microservice-demo/internal/http/response"
)

type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (auth.CurrentUser, error)
}

func Auth(verifier TokenVerifier, roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				response.Error(w, r, http.StatusUnauthorized, "AUTH_TOKEN_MISSING", "缺少登录凭证")
				return
			}
			user, err := verifier.VerifyToken(r.Context(), token)
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "登录凭证无效或已过期")
				return
			}
			if len(allowed) > 0 && !allowed[user.Role] {
				response.Error(w, r, http.StatusForbidden, "AUTH_FORBIDDEN", "无权限访问接口")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithCurrentUser(r.Context(), user)))
		})
	}
}

func bearerToken(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}
