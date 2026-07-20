package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("token inválido ou expirado")

const tokenTTL = 30 * 24 * time.Hour // 30 dias — a ideia é o login "ficar salvo"

type claims struct {
	UserID  string `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// Claims são as informações extraídas de um token já validado.
type Claims struct {
	UserID  uuid.UUID
	IsAdmin bool
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IssueToken emite um token para userID. isAdmin reflete o privilégio de quem
// está de fato agindo (importante para impersonação: ao "assumir" outro
// usuário, isAdmin continua refletindo o admin que iniciou a ação, não o
// usuário assumido).
func IssueToken(secret string, userID uuid.UUID, isAdmin bool) (string, error) {
	c := claims{
		UserID:  userID.String(),
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenString string) (Claims, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(c.UserID)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: userID, IsAdmin: c.IsAdmin}, nil
}
