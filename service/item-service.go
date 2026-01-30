package service

import "bpl/repository"

type ItemService struct {
	itemRepository *repository.ItemRepository
}

func NewItemService() *ItemService {
	return &ItemService{
		itemRepository: repository.NewItemRepository(),
	}
}

func (s *ItemService) SaveItems(itemNames []string, itemType string) error {
	items := make([]*repository.Item, 0, len(itemNames))
	for _, name := range itemNames {
		items = append(items, &repository.Item{Name: name, ItemType: itemType})
	}
	return s.itemRepository.SaveItems(items)
}

func (s *ItemService) SaveItem(itemName string, itemType string) (*repository.Item, error) {
	item := &repository.Item{Name: itemName, ItemType: itemType}
	return s.itemRepository.SaveItem(item)
}

func (s *ItemService) GetItemMap() (map[string]map[string]int, error) {
	return s.itemRepository.GetItemMap()
}
