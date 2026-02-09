package internal

import (
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{
				"error": "Invalid authorization format. Expected: Bearer {token}",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(401, gin.H{
				"error":   "Invalid token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(401, gin.H{
				"error": "Token is not valid",
			})
			c.Abort()
			return
		}

		if time.Now().Unix() > claims.ExpiresAt {
			c.JSON(401, gin.H{
				"error": "Token has expired",
			})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)

		c.Next()
	}
}
