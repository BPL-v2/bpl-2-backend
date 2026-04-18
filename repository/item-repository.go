package repository

import (
	"bpl/config"
	"fmt"

	"gorm.io/gorm"
)

type ItemType string

const (
	ItemTypeUnique ItemType = "unique"
	ItemTypeGem    ItemType = "gem"
)

type Item struct {
	Id       int      `gorm:"primaryKey;autoIncrement"`
	Name     string   `gorm:"not null"`
	ItemType ItemType `gorm:"not null"`
}

type ItemRepository interface {
	SaveItem(item *Item) (*Item, error)
	SaveItems(items []*Item) error
	GetItemMap() (map[ItemType]map[string]int, error)
}

type ItemRepositoryImpl struct {
	DB *gorm.DB
}

func NewItemRepository() ItemRepository {
	return &ItemRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ItemRepositoryImpl) SaveItem(item *Item) (*Item, error) {
	err := r.DB.
		Where(Item{Name: item.Name, ItemType: item.ItemType}).
		FirstOrCreate(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *ItemRepositoryImpl) SaveItems(items []*Item) error {
	return r.DB.Create(&items).Error
}

func (r *ItemRepositoryImpl) GetItemMap() (map[ItemType]map[string]int, error) {
	items := []*Item{}
	err := r.DB.Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching items: %w", err)
	}
	itemMap := make(map[ItemType]map[string]int)
	for _, item := range items {
		if _, ok := itemMap[item.ItemType]; !ok {
			itemMap[item.ItemType] = make(map[string]int)
		}
		itemMap[item.ItemType][item.Name] = item.Id
	}
	return itemMap, nil
}
