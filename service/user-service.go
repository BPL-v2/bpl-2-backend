package service

import (
	"bpl/auth"
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserService struct {
	UserRepository  *repository.UserRepository
	OauthRepository *repository.OauthRepository
}

func NewUserService() *UserService {
	return &UserService{
		UserRepository:  repository.NewUserRepository(),
		OauthRepository: repository.NewOauthRepository(),
	}
}

func (s *UserService) GetUserByDiscordId(discordId string) (*repository.User, error) {
	oauth, err := s.OauthRepository.GetOauthByProviderAndAccountID(repository.ProviderDiscord, discordId)
	if err != nil {
		return nil, err
	}
	return oauth.User, nil
}

func (s *UserService) GetUserByPoEAccount(poeAccount string) (*repository.User, error) {
	return s.UserRepository.GetUserByPoEAccount(poeAccount)
}

func (s *UserService) GetUserByTwitchId(twitchId string) (*repository.User, error) {
	return s.UserRepository.GetUserByTwitchId(twitchId)
}

func (s *UserService) SaveUser(user *repository.User) (*repository.User, error) {
	return s.UserRepository.SaveUser(user)
}

func (s *UserService) GetUsers(preloads ...string) ([]*repository.User, error) {
	return s.UserRepository.GetUsers(preloads...)
}

func (s *UserService) GetUserById(id int, preloads ...string) (*repository.User, error) {
	return s.UserRepository.GetUserById(id, preloads...)
}

func (s *UserService) GetUserFromAuthCookie(c *gin.Context) (*repository.User, error) {
	cookie, err := c.Cookie("auth")
	if err != nil {
		return nil, err
	}
	return s.GetUserFromToken(cookie)
}

func (s *UserService) GetUserFromToken(tokenString string) (*repository.User, error) {
	token, err := auth.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	claims := &auth.Claims{}
	if token.Valid {
		claims.FromJWTClaims(token.Claims)
		if err := claims.Valid(); err != nil {
			return nil, err
		}
		return s.GetUserById(claims.UserID, "OauthAccounts")
	}
	return nil, jwt.ErrInvalidKey
}

func (s *UserService) ChangePermissions(userId int, permissions []repository.Permission) (*repository.User, error) {
	user, err := s.GetUserById(userId)
	if err != nil {
		return nil, err
	}
	user.Permissions = permissions
	return s.UserRepository.SaveUser(user)
}

func (s *UserService) RemoveProvider(user *repository.User, provider repository.Provider) (*repository.User, error) {

	if len(user.OauthAccounts) < 2 {
		return nil, fmt.Errorf("cannot remove last provider")
	}

	for _, oauth := range user.OauthAccounts {
		if oauth.Provider == provider {

			s.OauthRepository.DB.Delete(oauth)
		}
	}
	return s.GetUserById(user.ID, "OauthAccounts")
}

func (s *UserService) DiscordServerCheck(user *repository.User) error {
	for _, oauth := range user.OauthAccounts {
		if oauth.Provider == repository.ProviderDiscord {
			memberIds, err := client.NewLocalDiscordClient().GetServerMemberIds()
			fmt.Println(memberIds)
			if err != nil || utils.Contains(memberIds, oauth.AccountID) {
				return nil
			} else {
				return fmt.Errorf("you have not joined the discord server")
			}
		}
	}
	return fmt.Errorf("you do not have a discord account linked")
}
