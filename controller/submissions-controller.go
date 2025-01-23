package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubmissionController struct {
	submissionService *service.SubmissionService
	userService       *service.UserService
	teamService       *service.TeamService
	eventService      *service.EventService
}

func NewSubmissionController(db *gorm.DB) *SubmissionController {
	return &SubmissionController{
		submissionService: service.NewSubmissionService(db),
		userService:       service.NewUserService(db),
		teamService:       service.NewTeamService(db),
		eventService:      service.NewEventService(db),
	}
}

func setupSubmissionController(db *gorm.DB) []RouteInfo {
	e := NewSubmissionController(db)
	baseUrl := "events/:event_id/submissions"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getSubmissionsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.submitBountyHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/:submission_id", HandlerFunc: e.deleteSubmissionHandler(), Authenticated: true},
		{Method: "PUT", Path: "/:submission_id/review", HandlerFunc: e.reviewSubmissionHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetSubmissions
// @Description Fetches all submissions for an event
// @Tags submission
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} SubmissionResponse
// @Router /events/{event_id}/submissions [get]
func (e *SubmissionController) getSubmissionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventById(eventId, "Teams")
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		submissions, err := e.submissionService.GetSubmissions(eventId)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		teamUsers, err := e.teamService.GetTeamUserMapForEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(submissions, func(submission *repository.Submission) *SubmissionResponse {
			return toSubmissionResponse(submission, teamUsers)
		}))
	}
}

// @id SubmitBounty
// @Description Submits a bounty for an event
// @Tags submission
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 201 {object} SubmissionResponse
// @Router /events/{event_id}/submissions [put]
func (e *SubmissionController) submitBountyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		var submissionCreate SubmissionCreate
		if err := c.BindJSON(&submissionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		submission := submissionCreate.toModel()
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		submission, err = e.submissionService.SaveSubmission(submission, user)

		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toSubmissionResponse(submission, nil))
	}
}

// @id DeleteSubmission
// @Description Deletes a submission
// @Tags submission
// @Produce json
// @Param event_id path int true "Event ID"
// @Param submission_id path int true "Submission ID"
// @Success 204
// @Router /events/{event_id}/submissions/{submission_id} [delete]
func (e *SubmissionController) deleteSubmissionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		submissionId, err := strconv.Atoi(c.Param("submission_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}

		err = e.submissionService.DeleteSubmission(submissionId, user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}

// @id ReviewSubmission
// @Description Reviews a submission
// @Tags submission
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param submission_id path int true "Submission ID"
// @Success 200 {object} SubmissionResponse
// @Router /events/{event_id}/submissions/{submission_id}/review [put]
func (e *SubmissionController) reviewSubmissionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		submissionId, err := strconv.Atoi(c.Param("submission_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}

		var submissionReview *SubmissionReview
		if err := c.BindJSON(&submissionReview); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		submission, err := e.submissionService.ReviewSubmission(submissionId, submissionReview.toModel(), user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toSubmissionResponse(submission, nil))
	}
}

type SubmissionCreate struct {
	ID          *int      `json:"id"`
	ObjectiveID int       `json:"objective_id" binding:"required"`
	Timestamp   time.Time `json:"timestamp" binding:"required"`
	Number      int       `json:"number"`
	Proof       string    `json:"proof"`
	Comment     string    `json:"comment"`
}

type SubmissionReview struct {
	ApprovalStatus repository.ApprovalStatus `json:"approval_status" binding:"required,oneof=APPROVED PENDING REJECTED"`
	ReviewComment  string                    `json:"review_comment"`
}

func (s *SubmissionCreate) toModel() *repository.Submission {
	submission := &repository.Submission{
		ObjectiveID: s.ObjectiveID,
		Number:      s.Number,
		Proof:       s.Proof,
		Timestamp:   s.Timestamp,
		Comment:     s.Comment,
	}
	if s.ID != nil {
		submission.ID = *s.ID
	}
	return submission
}

func (s *SubmissionReview) toModel() *repository.Submission {
	return &repository.Submission{
		ApprovalStatus: s.ApprovalStatus,
		ReviewComment:  &s.ReviewComment,
	}
}

type SubmissionResponse struct {
	ID             int                       `json:"id"`
	Objective      *ObjectiveResponse        `json:"objective"`
	Number         int                       `json:"number"`
	Proof          string                    `json:"proof"`
	Timestamp      time.Time                 `json:"timestamp"`
	ApprovalStatus repository.ApprovalStatus `json:"approval_status"`
	Comment        string                    `json:"comment"`
	User           *NonSensitiveUserResponse `json:"user"`
	TeamID         *int                      `json:"team_id"`
	ReviewComment  *string                   `json:"review_comment"`
	ReviewerID     *int                      `json:"reviewer_id"`
}

func toSubmissionResponse(submission *repository.Submission, teamUsers *map[int]int) *SubmissionResponse {
	response := &SubmissionResponse{
		ID:             submission.ID,
		Objective:      toObjectiveResponse(submission.Objective),
		Number:         submission.Number,
		Proof:          submission.Proof,
		Timestamp:      submission.Timestamp,
		ApprovalStatus: submission.ApprovalStatus,
		Comment:        submission.Comment,
		User:           toNonSensitiveUserResponse(submission.User),
		ReviewComment:  submission.ReviewComment,
		ReviewerID:     submission.ReviewerID,
	}
	if teamUsers != nil {
		teamID, ok := (*teamUsers)[submission.UserID]
		if ok {
			response.TeamID = &teamID
		}
	}
	return response
}
