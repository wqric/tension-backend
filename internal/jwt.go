package internal

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtSecret = []byte("your-secret-key")

type Claims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

func GenerateToken(userID int) (string, error) {

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "your-app-name",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
