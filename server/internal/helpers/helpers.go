package helpers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// float64ToNumeric converts *float64 to pgtype.Numeric.
func Float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*f); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

var (
	ErrEmptyOrMissingOrigin = errors.New("empty or missing origin")
)

func BuildRequestOriginCookie(name string, origin string, secure bool) (*http.Cookie, error) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return nil, ErrEmptyOrMissingOrigin
	}
	expires := time.Now().Add(10 * time.Minute)
	return &http.Cookie{
		Name:     name,
		Value:    origin,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	}, nil
}

func ReadRequestOriginCookie(name string, r *http.Request) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil || c == nil || c.Value == "" {
		return "", false
	}
	origin := strings.TrimSpace(c.Value)
	if origin == "" {
		return "", false
	}
	return origin, true
}

func DeleteRequestOriginCookie(name string, w gin.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}
