package service

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
)

type SubmissionService struct {
	submission_repository *repository.SubmissionRepository
}

func NewSubmissionService() *SubmissionService {
	return &SubmissionService{
		submission_repository: repository.NewSubmissionRepository(),
	}
}

func (e *SubmissionService) GetSubmissions(eventId int) ([]*repository.Submission, error) {
	objectiveService := NewObjectiveService()
	objectives, err := objectiveService.GetObjectivesByEventId(eventId)
	if err != nil {
		return nil, err
	}
	return e.submission_repository.GetSubmissionsForObjectives(objectives)
}

func (e *SubmissionService) SaveSubmission(submission *repository.Submission, submitter *repository.User) (*repository.Submission, error) {
	if submission.ID != 0 {
		existingSubmission, err := e.submission_repository.GetSubmissionById(submission.ID)
		if err != nil {
			return nil, err
		}
		if existingSubmission.UserID != submitter.ID {
			return nil, fmt.Errorf("you are not allowed to edit this submission")
		}

		existingSubmission, err = e.submission_repository.RemoveMatchFromSubmission(existingSubmission)
		if err != nil {
			return nil, err
		}
		existingSubmission.ObjectiveID = submission.ObjectiveID
		existingSubmission.Timestamp = submission.Timestamp
		existingSubmission.Number = submission.Number
		existingSubmission.Proof = submission.Proof
		existingSubmission.Comment = submission.Comment
		if existingSubmission.ApprovalStatus == repository.APPROVED {
			existingSubmission.ApprovalStatus = repository.PENDING
		}
		return e.submission_repository.SaveSubmission(existingSubmission)
	}
	submission.ApprovalStatus = repository.PENDING
	submission.User = submitter
	submission.UserID = submitter.ID
	return e.submission_repository.SaveSubmission(submission)
}

func (e *SubmissionService) ReviewSubmission(submissionId int, submissionReview *repository.Submission, reviewer *repository.User) (*repository.Submission, error) {
	submission, err := e.submission_repository.GetSubmissionById(submissionId)

	if err != nil {
		return nil, err
	}
	if !utils.Contains(reviewer.Permissions, "admin") {
		return nil, fmt.Errorf("you are not allowed to review submissions")
	}

	if submission.MatchID != nil {
		if submissionReview.ApprovalStatus != repository.APPROVED {
			submission, err = e.submission_repository.RemoveMatchFromSubmission(submission)
			if err != nil {
				return nil, err
			}
		}

	} else {
		if submissionReview.ApprovalStatus == repository.APPROVED {
			match := submission.ToObjectiveMatch()
			submission.Match = match
		}
	}
	submission.ApprovalStatus = submissionReview.ApprovalStatus
	submission.ReviewComment = submissionReview.ReviewComment
	submission.ReviewerID = &reviewer.ID
	return e.submission_repository.SaveSubmission(submission)
}

func (e *SubmissionService) DeleteSubmission(id int, user *repository.User) error {
	submission, err := e.submission_repository.GetSubmissionById(id)
	if err != nil {
		return err
	}
	if submission.UserID != user.ID {
		return fmt.Errorf("you are not allowed to delete this submission")
	}
	return e.submission_repository.DeleteSubmission(id)
}
