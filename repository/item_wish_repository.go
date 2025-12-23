package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type ItemWishRepository struct {
	DB *gorm.DB
}

func NewItemWishRepository() *ItemWishRepository {
	return &ItemWishRepository{DB: config.DatabaseConnection()}
}

type ItemWish struct {
	Id            int       `gorm:"not null;primaryKey;autoIncrement"`
	UserID        int       `gorm:"not null;index:idx_user_event_item_wish"`
	EventID       int       `gorm:"not null;index:idx_user_event_item_wish"`
	ItemField     ItemField `gorm:"not null"`
	Value         string    `gorm:"not null"`
	Fulfilled     bool      `gorm:"not null;default:false"`
	BuildEnabling bool      `gorm:"not null;default:false"`
	Priority      int       `gorm:"not null;default:0"`

	User  *User  `gorm:"foreignKey:UserID"`
	Event *Event `gorm:"foreignKey:EventID"`
}

func (r *ItemWishRepository) SaveItemWish(itemWish *ItemWish) (*ItemWish, error) {
	err := r.DB.Save(itemWish).Error
	return itemWish, err
}

func (r *ItemWishRepository) GetItemWishesForEventAndUser(eventId int, userId int) (itemWishes []*ItemWish, err error) {
	err = r.DB.Where("event_id = ? AND user_id = ?", eventId, userId).Find(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepository) GetItemWishesForTeam(teamId int) (itemWishes []*ItemWish, err error) {
	query := `SELECT iw.* FROM item_wishes iw
				JOIN teams t ON iw.event_id = t.event_id
				JOIN team_users tu ON t.id = tu.team_id
				WHERE t.id = ? AND tu.user_id = iw.user_id`
	err = r.DB.Raw(query, teamId).Scan(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepository) GetSimilarItemWishesInTeam(teamId int, itemField ItemField, value string) (itemWishes []*ItemWish, err error) {
	query := `SELECT iw.* FROM item_wishes iw
				JOIN teams t ON iw.event_id = t.event_id
				JOIN team_users tu ON t.id = tu.team_id
				WHERE t.id = ? AND iw.item_field = ? AND iw.value = ?`
	err = r.DB.Raw(query, teamId, itemField, value).Scan(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepository) DeleteItemWish(id int) error {
	return r.DB.Delete(&ItemWish{}, id).Error
}

func (r *ItemWishRepository) GetItemWishById(id int) (*ItemWish, error) {
	var itemWish ItemWish
	err := r.DB.First(&itemWish, id).Error
	if err != nil {
		return nil, err
	}
	return &itemWish, nil
}
