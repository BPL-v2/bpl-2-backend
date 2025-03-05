package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type SubmissionController struct {
	submissionService *service.SubmissionService
	userService       *service.UserService
	teamService       *service.TeamService
	eventService      *service.EventService
}

func NewSubmissionController() *SubmissionController {
	return &SubmissionController{
		submissionService: service.NewSubmissionService(),
		userService:       service.NewUserService(),
		teamService:       service.NewTeamService(),
		eventService:      service.NewEventService(),
	}
}

func setupSubmissionController() []RouteInfo {
	e := NewSubmissionController()
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
// @Param event_id path int true "Event Id"
// @Success 200 {array} Submission
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
		c.JSON(200, utils.Map(submissions, func(submission *repository.Submission) *Submission {
			return toSubmissionResponse(submission, teamUsers)
		}))
	}
}

// @id SubmitBounty
// @Description Submits a bounty for an event
// @Tags submission
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body SubmissionCreate true "Submission to create"
// @Success 201 {object} Submission
// @Router /events/{event_id}/submissions [put]
func (e *SubmissionController) submitBountyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		var submissionCreate SubmissionCreate
		if err := c.BindJSON(&submissionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		submission := submissionCreate.toModel()
		submission.EventId, _ = strconv.Atoi(c.Param("event_id"))
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		_, err = e.teamService.GetTeamForUser(submission.EventId, user.Id)
		if err != nil {
			c.JSON(403, gin.H{"error": "User does not participate in event"})
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
// @Param event_id path int true "Event Id"
// @Param submission_id path int true "Submission Id"
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
// @Param event_id path int true "Event Id"
// @Param submission_id path int true "Submission Id"
// @Param submission body SubmissionReview true "Submission review"
// @Success 200 {object} Submission
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
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var submissionReview *SubmissionReview
		if err := c.BindJSON(&submissionReview); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		model := submissionReview.toModel()
		model.EventId = eventId

		submission, err := e.submissionService.ReviewSubmission(submissionId, model, user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toSubmissionResponse(submission, nil))
	}
}

type SubmissionCreate struct {
	Id          *int      `json:"id"`
	ObjectiveId int       `json:"objective_id" binding:"required"`
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
		ObjectiveId: s.ObjectiveId,
		Number:      s.Number,
		Proof:       s.Proof,
		Timestamp:   s.Timestamp,
		Comment:     s.Comment,
	}
	if s.Id != nil {
		submission.Id = *s.Id
	}
	return submission
}

func (s *SubmissionReview) toModel() *repository.Submission {
	return &repository.Submission{
		ApprovalStatus: s.ApprovalStatus,
		ReviewComment:  &s.ReviewComment,
	}
}

type Submission struct {
	Id             int                       `json:"id" binding:"required"`
	Objective      *Objective                `json:"objective"`
	Number         int                       `json:"number" binding:"required"`
	Proof          string                    `json:"proof" binding:"required"`
	Timestamp      time.Time                 `json:"timestamp" binding:"required"`
	ApprovalStatus repository.ApprovalStatus `json:"approval_status" binding:"required"`
	Comment        string                    `json:"comment" binding:"required"`
	User           *NonSensitiveUser         `json:"user"`
	TeamId         *int                      `json:"team_id"`
	ReviewComment  *string                   `json:"review_comment"`
	ReviewerId     *int                      `json:"reviewer_id"`
}

func toSubmissionResponse(submission *repository.Submission, teamUsers *map[int]int) *Submission {
	response := &Submission{
		Id:             submission.Id,
		Objective:      toObjectiveResponse(submission.Objective),
		Number:         submission.Number,
		Proof:          submission.Proof,
		Timestamp:      submission.Timestamp,
		ApprovalStatus: submission.ApprovalStatus,
		Comment:        submission.Comment,
		User:           toNonSensitiveUserResponse(submission.User),
		ReviewComment:  submission.ReviewComment,
		ReviewerId:     submission.ReviewerId,
	}
	if teamUsers != nil {
		teamId, ok := (*teamUsers)[submission.UserId]
		if ok {
			response.TeamId = &teamId
		}
	}
	return response
}
