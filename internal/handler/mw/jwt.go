package mw

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	hoursInADay = 24
	splitSize   = 2
)

var secretKey []byte

type userCtxKeyType int

const userCtxKey userCtxKeyType = iota

type customClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func SetSecretKey(key []byte) {
	secretKey = key
}

func GenerateJWT(userID int, username string) (string, error) {
	claims := customClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(hoursInADay * time.Hour)), // вместо 24
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(secretKey) == 0 {
			http.Error(w, `{"errors":"jwt secret not configured"}`, http.StatusInternalServerError)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"errors":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", splitSize) // вместо 2
		if len(parts) != splitSize || parts[0] != "Bearer" {
			http.Error(w, `{"errors":"invalid token format"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		token, err := jwt.ParseWithClaims(tokenStr, &customClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrNoCookie
			}
			return secretKey, nil
		})
		if err != nil {
			http.Error(w, `{"errors":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(*customClaims)
		if !ok || !token.Valid {
			http.Error(w, `{"errors":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MustGetUserID(ctx context.Context) int {
	val := ctx.Value(userCtxKey)
	if val == nil {
		return 0
	}
	return val.(int)
}
