package service

import (
	"bpl/client"
	"bpl/repository"
	"fmt"
	"sync"

	"github.com/lib/pq"
)

type ItemService struct {
	itemRepository *repository.ItemRepository
	itemMap        map[repository.ItemType]map[string]int
	mu             sync.RWMutex
}

func NewItemService() *ItemService {
	return &ItemService{
		itemRepository: repository.NewItemRepository(),
		itemMap:        make(map[repository.ItemType]map[string]int),
	}
}

func (s *ItemService) SaveItems(itemNames []string, itemType repository.ItemType) error {
	items := make([]*repository.Item, 0, len(itemNames))
	for _, name := range itemNames {
		items = append(items, &repository.Item{Name: name, ItemType: itemType})
	}
	return s.itemRepository.SaveItems(items)
}

func (s *ItemService) SaveItem(itemName string, itemType repository.ItemType) (*repository.Item, error) {
	item := &repository.Item{Name: itemName, ItemType: itemType}
	return s.itemRepository.SaveItem(item)
}

func (s *ItemService) GetIds(items []*repository.Item) (pq.Int32Array, error) {
	itemIds := make(pq.Int32Array, 0)
	_, err := s.GetItemMap()
	if err != nil {
		return itemIds, err
	}
	for _, item := range items {
		itemId, err := s.GetOrCreateId(item.Name, item.ItemType)
		if err != nil {
			return itemIds, err
		}
		itemIds = append(itemIds, int32(itemId))
	}
	return itemIds, nil
}

func (s *ItemService) GetOrCreateId(itemName string, itemType repository.ItemType) (int, error) {
	s.mu.RLock()
	itemTypeMap, ok := s.itemMap[itemType]
	if ok {
		itemId, ok := itemTypeMap[itemName]
		if ok {
			s.mu.RUnlock()
			return itemId, nil
		}
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.itemMap[itemType] == nil {
		s.itemMap[itemType] = make(map[string]int)
	}

	itemId, ok := s.itemMap[itemType][itemName]
	if ok {
		return itemId, nil
	}

	savedItem, err := s.SaveItem(itemName, itemType)
	if err != nil {
		return 0, fmt.Errorf("error saving item %s: %w", itemName, err)
	}

	s.itemMap[itemType][itemName] = savedItem.Id
	return savedItem.Id, nil
}

func (s *ItemService) GetItemMap() (map[repository.ItemType]map[string]int, error) {
	s.mu.RLock()
	if len(s.itemMap) > 0 {
		defer s.mu.RUnlock()
		return s.itemMap, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.itemMap) == 0 {
		itemMap, err := s.itemRepository.GetItemMap()
		if err != nil {
			return nil, err
		}
		s.itemMap = itemMap
	}
	return s.itemMap, nil
}

func (c *ItemService) GetItemIds(character *client.Character) (pq.Int32Array, error) {
	itemIds := make(pq.Int32Array, 0)
	items := make([]*repository.Item, 0)
	for _, item := range character.GetAllItems() {
		if item.Rarity != nil && *item.Rarity == "Unique" {
			items = append(items, &repository.Item{Name: item.Name, ItemType: repository.ItemTypeUnique})
		}
		if item.FrameType != nil && *item.FrameType == 4 {
			items = append(items, &repository.Item{Name: item.TypeLine, ItemType: repository.ItemTypeGem})
		}
	}
	if len(items) > 0 {
		return c.GetIds(items)
	}
	return itemIds, nil
}
