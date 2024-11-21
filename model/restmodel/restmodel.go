package restmodel

type Event struct {
	ID                int    `json:"id"`
	Name              string `json:"name" binding:"required"`
	ScoringCategoryID int    `json:"scoring_category_id" `
}
