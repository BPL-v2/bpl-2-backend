package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"math"
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

func (s *ItemWishService) CreateItemWish(itemWish *repository.ItemWish, teamId int) (*repository.ItemWish, error) {
	itemWishes, err := s.itemWishRepository.GetSimilarItemWishesInTeam(teamId, itemWish.ItemField, itemWish.Value)
	if err != nil {
		return nil, err
	}
	itemWish.Priority = len(itemWishes)
	return s.itemWishRepository.SaveItemWish(itemWish)
}

func (s *ItemWishService) UpdateItemWish(itemWish *repository.ItemWish, teamId int, Fulfilled *bool, BuildEnabling *bool, Priority *int) (*repository.ItemWish, error) {
	if Fulfilled != nil {
		itemWish.Fulfilled = *Fulfilled
	}
	if BuildEnabling != nil {
		itemWish.BuildEnabling = *BuildEnabling
	}
	if Priority != nil {
		itemWishes, err := s.itemWishRepository.GetSimilarItemWishesInTeam(teamId, itemWish.ItemField, itemWish.Value)
		if err != nil {
			return nil, err
		}
		priority := int(math.Max(math.Min(float64(*Priority), float64(len(itemWishes))), 0))
		for _, iw := range itemWishes {
			if iw.Priority == priority {
				iw.Priority = itemWish.Priority
				s.itemWishRepository.SaveItemWish(iw)
				break
			}
		}
		itemWish.Priority = priority
	}
	return s.itemWishRepository.SaveItemWish(itemWish)
}

func (s *ItemWishService) GetItemWishById(id int) (*repository.ItemWish, error) {
	return s.itemWishRepository.GetItemWishById(id)
}

func (s *ItemWishService) DeleteItemWish(id int) error {
	return s.itemWishRepository.DeleteItemWish(id)
}

func (s *ItemWishService) GetItemWishesForTeam(teamId int) ([]*repository.ItemWish, error) {
	itemWishes, err := s.itemWishRepository.GetItemWishesForTeam(teamId)
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (s *ItemWishService) UpdateItemWishFulfillment(teamId int, userId int, character *client.Character) error {
	itemWishes, err := s.itemWishRepository.GetItemWishesForTeamAndUser(teamId, userId)
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
