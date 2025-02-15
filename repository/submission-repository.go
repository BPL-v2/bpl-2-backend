package repository

import (
	"bpl/config"
	"bpl/utils"
	"time"

	"gorm.io/gorm"
)

type ApprovalStatus = string

const (
	APPROVED ApprovalStatus = "APPROVED"
	REJECTED ApprovalStatus = "REJECTED"
	PENDING  ApprovalStatus = "PENDING"
)

type Submission struct {
	ID             int            `gorm:"primaryKey"`
	ObjectiveID    int            `gorm:"not null;references:objectives(id)"`
	Timestamp      time.Time      `gorm:"not null"`
	Number         int            `gorm:"not null"`
	UserID         int            `gorm:"not null;references:users(id)"`
	Proof          string         `gorm:"not null"`
	Comment        string         `gorm:"not null"`
	ApprovalStatus ApprovalStatus `gorm:"not null;type:bpl2.approval_status"`
	ReviewComment  *string        `gorm:"null"`
	ReviewerID     *int           `gorm:"null;references:users(id)"`
	MatchID        *int           `gorm:"null;references:objective_matches(id)"`
	EventID        int            `gorm:"not null;references:events(id)"`

	Match     *ObjectiveMatch `gorm:"foreignKey:MatchID;constraint:OnDelete:CASCADE;"`
	Objective *Objective      `gorm:"foreignKey:ObjectiveID;constraint:OnDelete:CASCADE;"`
	User      *User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Reviewer  *User           `gorm:"foreignKey:ReviewerID;constraint:OnDelete:CASCADE;"`
}

func (s *Submission) ToObjectiveMatch() *ObjectiveMatch {
	return &ObjectiveMatch{
		ObjectiveID: s.ObjectiveID,
		Timestamp:   s.Timestamp,
		Number:      s.Number,
		UserID:      s.UserID,
		EventId:     s.EventID,
	}
}

type SubmissionRepository struct {
	DB *gorm.DB
}

func NewSubmissionRepository() *SubmissionRepository {
	return &SubmissionRepository{DB: config.DatabaseConnection()}
}

func (r *SubmissionRepository) GetSubmissionsForObjectives(objectives []*Objective) ([]*Submission, error) {
	var submissions []*Submission
	result := r.DB.Preload("Objective").Preload("User").Find(&submissions, "objective_id IN ?", utils.Map(objectives, func(o *Objective) int { return o.ID }))
	if result.Error != nil {
		return nil, result.Error
	}
	return submissions, nil
}

func (r *SubmissionRepository) GetSubmissionById(id int) (*Submission, error) {
	var submission Submission
	result := r.DB.First(&submission, "id = ?", id).Preload("Objective")
	if result.Error != nil {
		return nil, result.Error
	}
	return &submission, nil
}

func (r *SubmissionRepository) SaveSubmission(submission *Submission) (*Submission, error) {
	result := r.DB.Save(submission)
	if result.Error != nil {
		return nil, result.Error
	}
	return submission, nil
}

func (r *SubmissionRepository) RemoveMatchFromSubmission(submission *Submission) (*Submission, error) {
	if submission.MatchID == nil {
		return submission, nil
	}
	tx := r.DB.Begin()
	matchId := *submission.MatchID
	submission.MatchID = nil
	result := r.DB.Save(submission)
	if result.Error != nil {
		tx.Rollback()
		return submission, result.Error
	}
	result = r.DB.Delete(&ObjectiveMatch{}, "id = ?", matchId)
	if result.Error != nil {
		tx.Rollback()
		return submission, result.Error
	}
	tx.Commit()
	return submission, nil

}

func (r *SubmissionRepository) DeleteSubmission(submissionId int) error {
	result := r.DB.Delete(&Submission{}, "id = ?", submissionId)
	return result.Error
}
