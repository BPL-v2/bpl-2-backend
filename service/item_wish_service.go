package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
)

type ItemWishService struct {
	itemWishRepository *repository.ItemWishRepository
	teamRepository     *repository.TeamRepository
}

func NewItemWishService() *ItemWishService {
	return &ItemWishService{
		itemWishRepository: repository.NewItemWishRepository(),
	}
}

func (s *ItemWishService) SaveItemWish(itemWish *repository.ItemWish) (*repository.ItemWish, error) {
	return s.itemWishRepository.SaveItemWish(itemWish)
}

func (s *ItemWishService) GetItemWishById(id int) (*repository.ItemWish, error) {
	return s.itemWishRepository.GetItemWishById(id)
}

func (s *ItemWishService) DeleteItemWish(id int) error {
	return s.itemWishRepository.DeleteItemWish(id)
}

func (s *ItemWishService) GetItemWishesForTeam(eventId int, teamId int) ([]*repository.ItemWish, error) {
	itemWishes, err := s.itemWishRepository.GetItemWishesForTeam(eventId, teamId)
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (s *ItemWishService) UpdateItemWishFulfillment(eventId int, userId int, character *client.Character) error {
	itemWishes, err := s.itemWishRepository.GetItemWishesForEventAndUser(eventId, userId)
	if err != nil {
		return err
	}
	for _, itemWish := range itemWishes {
		if itemWish.Fulfilled {
			continue
		}
		for _, item := range utils.FlatMap(*character.Equipment, func(i client.Item) []client.Item {
			return *i.SocketedItems
		}) {
			switch itemWish.ItemField {
			case repository.BASE_TYPE:
				if item.BaseType == itemWish.Value {
					itemWish.Fulfilled = true
					s.itemWishRepository.SaveItemWish(itemWish)
				}
			case repository.NAME:
				if item.Name == itemWish.Value {
					itemWish.Fulfilled = true
					s.itemWishRepository.SaveItemWish(itemWish)
				}
			default:
				continue
			}
		}
	}
	return nil
}
