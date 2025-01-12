
package auth

import (
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	jwtManager *JWTManager
}

func NewAuthMiddleware(jwtManager *JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}


func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]


		_, err := am.jwtManager.Validate(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token: "+err.Error(), http.StatusUnauthorized)
			return
		}


		next.ServeHTTP(w, r)
	})
}
