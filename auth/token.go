package auth

import (
	"os"
	"time"

	"bpl/repository"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte(os.Getenv("JWT_SECRET"))

type Claims struct {
	UserID      int      `json:"user_id"`
	Permissions []string `json:"permissions"`
	DiscordID   int64    `json:"discord_id"`
	AccountName string   `json:"account_name"`
	DiscordName string   `json:"discord_name"`
	Exp         int64    `json:"exp"`
}

func (claims *Claims) FromJWTClaims(jwtClaims jwt.Claims) {
	claims.UserID = jwtClaims.(jwt.MapClaims)["user_id"].(int)
	claims.Permissions = jwtClaims.(jwt.MapClaims)["permissions"].([]string)
	claims.DiscordID = jwtClaims.(jwt.MapClaims)["discord_id"].(int64)
	claims.AccountName = jwtClaims.(jwt.MapClaims)["account_name"].(string)
	claims.Exp = int64(jwtClaims.(jwt.MapClaims)["exp"].(float64))
	claims.DiscordName = jwtClaims.(jwt.MapClaims)["discord_name"].(string)
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
			"user_id":      user.ID,
			"account_name": user.AccountName,
			"discord_name": user.DiscordName,
			"discord_id":   user.DiscordID,
			"permissions":  user.Permissions,
			"exp":          time.Now().Add(time.Hour * 24 * 7).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}
	return token, nil
}
