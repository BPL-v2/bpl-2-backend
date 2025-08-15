package auth

import (
	"time"

	"bpl/config"
	"bpl/repository"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserId      int      `json:"user_id"`
	Permissions []string `json:"permissions"`
	Exp         int64    `json:"exp"`
}

func (claims *Claims) FromJWTClaims(jwtClaims jwt.Claims) {
	permissions := []string{}
	if jwtClaims.(jwt.MapClaims)["permissions"] != nil {
		for _, perm := range jwtClaims.(jwt.MapClaims)["permissions"].([]interface{}) {
			permissions = append(permissions, perm.(string))
		}
	}
	claims.Permissions = permissions
	claims.UserId = int(jwtClaims.(jwt.MapClaims)["user_id"].(float64))
	claims.Exp = int64(jwtClaims.(jwt.MapClaims)["exp"].(float64))
}

func (claims *Claims) Valid() error {
	if time.Now().Unix() > claims.Exp {
		return jwt.ErrTokenExpired
	}
	return nil
}

func CreateToken(user *repository.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"user_id":     user.Id,
			"permissions": user.Permissions,
			"exp":         time.Now().Add(time.Hour * 24 * 21).Unix(),
		})

	tokenString, err := token.SignedString([]byte(config.Env().JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Env().JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}
	return token, nil
}
