package service

import (
	"bpl/auth"
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserService struct {
	userRepository  *repository.UserRepository
	oauthRepository *repository.OauthRepository
	teamService     *TeamService
}

func NewUserService() *UserService {
	return &UserService{
		userRepository:  repository.NewUserRepository(),
		oauthRepository: repository.NewOauthRepository(),
		teamService:     NewTeamService(),
	}
}

func (s *UserService) GetUserByDiscordId(discordId string) (*repository.User, error) {
	oauth, err := s.oauthRepository.GetOauthByProviderAndAccountID(repository.ProviderDiscord, discordId)
	if err != nil {
		return nil, err
	}
	return oauth.User, nil
}

func (s *UserService) GetUserByPoEAccount(poeAccount string) (*repository.User, error) {
	return s.userRepository.GetUserByPoEAccount(poeAccount)
}

func (s *UserService) GetUserByTwitchId(twitchId string) (*repository.User, error) {
	return s.userRepository.GetUserByTwitchId(twitchId)
}

func (s *UserService) SaveUser(user *repository.User) (*repository.User, error) {
	return s.userRepository.SaveUser(user)
}

func (s *UserService) GetAllUsers(preloads ...string) ([]*repository.User, error) {
	users, err := s.userRepository.GetAllUsers()
	if err != nil {
		return nil, err
	}
	if len(preloads) > 0 && utils.Contains(preloads, "OauthAccounts") {
		oauths, err := s.oauthRepository.GetAllOauths()
		if err != nil {
			return nil, err
		}
		userOauthMap := make(map[int][]*repository.Oauth)
		for _, oauth := range oauths {
			if _, ok := userOauthMap[oauth.UserID]; !ok {
				userOauthMap[oauth.UserID] = []*repository.Oauth{}
			}
			userOauthMap[oauth.UserID] = append(userOauthMap[oauth.UserID], oauth)
		}
		for _, user := range users {
			if oauths, ok := userOauthMap[user.ID]; ok {
				user.OauthAccounts = oauths
			}
		}
	}
	return users, nil
}

func (s *UserService) GetUserById(id int, preloads ...string) (*repository.User, error) {
	return s.userRepository.GetUserById(id, preloads...)
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
	return s.userRepository.SaveUser(user)
}

func (s *UserService) RemoveProvider(user *repository.User, provider repository.Provider) (*repository.User, error) {

	if len(user.OauthAccounts) < 2 {
		return nil, fmt.Errorf("cannot remove last provider")
	}

	for _, oauth := range user.OauthAccounts {
		if oauth.Provider == provider {

			s.oauthRepository.DB.Delete(oauth)
		}
	}
	return s.GetUserById(user.ID, "OauthAccounts")
}

func (s *UserService) DiscordServerCheck(user *repository.User) error {
	for _, oauth := range user.OauthAccounts {
		if oauth.Provider == repository.ProviderDiscord {
			memberIds, err := client.NewLocalDiscordClient().GetServerMemberIds()
			if err != nil || utils.Contains(memberIds, oauth.AccountID) {
				return nil
			} else {
				return fmt.Errorf("you have not joined the discord server")
			}
		}
	}
	return fmt.Errorf("you do not have a discord account linked")
}

func (s *UserService) AddUserFromStashchange(userName string, event *repository.Event) (*repository.User, error) {
	// should only be used for testing
	user := &repository.User{
		DisplayName: userName,
		Permissions: []repository.Permission{},
	}
	u, err := s.SaveUser(user)
	if err != nil {
		return nil, err
	}
	oauth := &repository.Oauth{
		UserID:      u.ID,
		Provider:    repository.ProviderPoE,
		AccessToken: "dummy",
		AccountID:   userName,
		Name:        userName,
		Expiry:      time.Now(),
	}
	err = s.oauthRepository.DB.Save(oauth).Error
	if err != nil {
		return nil, err
	}
	u.OauthAccounts = append(u.OauthAccounts, oauth)
	team := event.Teams[rand.IntN(len(event.Teams))]
	s.teamService.AddUsersToTeams([]*repository.TeamUser{{TeamID: team.ID, UserID: u.ID}}, event)
	return u, nil
}

func (s *UserService) GetUsersForEvent(eventId int) ([]*repository.TeamUserWithPoEAccountName, error) {
	return s.userRepository.GetUsersForEvent(eventId)
}
