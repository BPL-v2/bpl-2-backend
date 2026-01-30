package repository

import (
	"bpl/config"
	"fmt"

	"gorm.io/gorm"
)

type Item struct {
	Id   int    `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"not null"`
}

type ItemRepository struct {
	DB *gorm.DB
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{DB: config.DatabaseConnection()}
}

func (r *ItemRepository) SaveItem(item *Item) error {
	return r.DB.Create(&item).Error
}

func (r *ItemRepository) SaveItems(items []*Item) error {
	return r.DB.Create(&items).Error
}

func (r *ItemRepository) GetItemMap() (map[string]int, error) {
	items := []*Item{}
	err := r.DB.Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching items: %w", err)
	}
	itemMap := make(map[string]int)
	for _, item := range items {
		itemMap[item.Name] = item.Id
	}
	return itemMap, nil
}
