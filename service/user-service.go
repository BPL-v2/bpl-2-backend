package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type UserService struct {
	UserRepository *repository.UserRepository
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		UserRepository: repository.NewUserRepository(db),
	}
}

func (s *UserService) GetOrCreateUserByDiscordId(discordId int64) (*repository.User, error) {
	user, err := s.UserRepository.GetUserByDiscordId(discordId)
	if err != nil {
		user = &repository.User{DiscordID: discordId}
		user, err = s.UserRepository.SaveUser(user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}
