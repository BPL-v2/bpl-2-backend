package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"fmt"
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
		{Method: "PUT", Path: "/:submission_id/review", HandlerFunc: e.reviewSubmissionHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionSubmissionJudge}},
		{Method: "PUT", Path: "/admin", HandlerFunc: e.setBulkSubmissionForAdmin(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionSubmissionJudge}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id SetBulkSubmissionForAdmin
// @Description Sets submissions for teams
// @Tags submission
// @Accept json
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body TeamSubmissionCreate true "Submissions to create"
// @Success 201 {array} Submission
// @Router /events/{event_id}/submissions/admin [put]
func (e *SubmissionController) setBulkSubmissionForAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		var submissionCreate TeamSubmissionCreate
		if err := c.BindJSON(&submissionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		teamLeads, err := e.teamService.GetTeamLeadsForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		submissions, err := e.submissionService.SaveBulkSubmissions(submissionCreate.toModels(event.Id, user.Id, teamLeads))
		if err != nil {
			fmt.Println(err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, utils.Map(submissions, func(submission *repository.Submission) *Submission {
			return toSubmissionResponse(submission, nil)
		},
		))

	}
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
		event := getEvent(c)
		if event == nil {
			return
		}
		submissions, err := e.submissionService.GetSubmissions(event.Id)
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
// @Security BearerAuth
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
		user, err := e.userService.GetUserFromAuthHeader(c)
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
// @Security BearerAuth
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
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		submission, err := e.submissionService.GetSubmissionById(submissionId)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		if submission.UserId != user.Id && !utils.Contains(user.Permissions, repository.PermissionSubmissionJudge) {
			c.JSON(403, gin.H{"error": "You are not allowed to delete this submission"})
			return
		}
		err = e.submissionService.DeleteSubmission(submission, user)
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
// @Security BearerAuth
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
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}

		var submissionReview *SubmissionReview
		if err := c.BindJSON(&submissionReview); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		model := submissionReview.toModel()
		model.EventId = event.Id

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

type TeamSubmissionCreate struct {
	ObjectiveId int   `json:"objective_id" binding:"required"`
	TeamIds     []int `json:"team_ids" binding:"required"`
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

func (s *TeamSubmissionCreate) toModels(eventId int, reviewerId int, teamLeads map[int][]*repository.TeamUser) []*repository.Submission {
	now := time.Now()
	submissions := make([]*repository.Submission, 0)
	for place, teamId := range s.TeamIds {
		submissions = append(submissions, &repository.Submission{
			ObjectiveId:    s.ObjectiveId,
			Timestamp:      now.Add(time.Duration(place) * time.Second),
			Number:         1,
			UserId:         teamLeads[teamId][0].UserId,
			ApprovalStatus: repository.PENDING,
			ReviewerId:     &reviewerId,
			EventId:        eventId,
		})
	}
	return submissions
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
