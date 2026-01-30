package repository

import (
	"bpl/config"
	"fmt"

	"gorm.io/gorm"
)

type Item struct {
	Id       int    `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"not null"`
	ItemType string `gorm:"not null"`
}

type ItemRepository struct {
	DB *gorm.DB
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{DB: config.DatabaseConnection()}
}

func (r *ItemRepository) SaveItem(item *Item) (*Item, error) {
	if err := r.DB.Create(&item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *ItemRepository) SaveItems(items []*Item) error {
	return r.DB.Create(&items).Error
}

func (r *ItemRepository) GetItemMap() (map[string]map[string]int, error) {
	items := []*Item{}
	err := r.DB.Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching items: %w", err)
	}
	itemMap := make(map[string]map[string]int)
	for _, item := range items {
		if _, ok := itemMap[item.ItemType]; !ok {
			itemMap[item.ItemType] = make(map[string]int)
		}
		itemMap[item.ItemType][item.Name] = item.Id
	}
	return itemMap, nil
}
