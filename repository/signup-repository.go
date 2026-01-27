package repository

import (
	"bpl/config"
	"bpl/metrics"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Signup struct {
	EventId          int       `gorm:"not null;references:event(id);primaryKey"`
	UserId           int       `gorm:"not null;references:event(id);primaryKey"`
	Timestamp        time.Time `gorm:"not null"`
	User             *User     `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
	PartnerWish      *string   `gorm:"null"`
	ExpectedPlayTime int       `gorm:"not null"`
	NeedsHelp        bool      `gorm:"not null"`
	WantsToHelp      bool      `gorm:"not null"`
	ActualPlayTime   int       `gorm:"not null;default:0"`
	Extra            *string   `gorm:"null"`
}

func GetSignupPartners(signups []*Signup) map[int]*Signup {
	partnerMap := make(map[int]*Signup)
	for _, signup1 := range signups {
		if signup1.PartnerWish != nil {
			for _, signup2 := range signups {
				if signup2.User.HasPoEName(*signup1.PartnerWish) {
					partnerMap[signup1.User.Id] = signup2
				}
			}
		}
	}
	return partnerMap
}

func PoENameWithoutDiscriminator(name *string) string {
	if name == nil {
		return ""
	}
	return strings.ToLower(strings.Split(*name, "#")[0])
}

type SignupRepository struct {
	DB *gorm.DB
}

func NewSignupRepository() *SignupRepository {
	return &SignupRepository{DB: config.DatabaseConnection()}
}

func (r *SignupRepository) SaveSignup(signup *Signup) (*Signup, error) {
	result := r.DB.Save(signup)
	if result.Error != nil {
		return nil, result.Error
	}
	return signup, nil
}

func (r *SignupRepository) RemoveSignupForUser(userId int, eventId int) error {
	result := r.DB.Delete(&Signup{}, &Signup{UserId: userId, EventId: eventId})
	return result.Error
}

func (r *SignupRepository) GetSignupForUser(userId int, eventId int) (*Signup, error) {
	signup := Signup{}
	result := r.DB.First(&signup, &Signup{UserId: userId, EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return &signup, nil
}

func (r *SignupRepository) GetSignupsForEvent(eventId int) ([]*Signup, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetSignupsForEvent"))
	defer timer.ObserveDuration()
	signups := make([]*Signup, 0)
	result := r.DB.Preload("User").Preload("User.OauthAccounts").Order("timestamp ASC").Find(&signups, &Signup{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return signups, nil
}
