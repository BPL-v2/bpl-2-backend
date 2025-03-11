package service

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
)

type SubmissionService struct {
	submissionRepository *repository.SubmissionRepository
}

func NewSubmissionService() *SubmissionService {
	return &SubmissionService{
		submissionRepository: repository.NewSubmissionRepository(),
	}
}

func (e *SubmissionService) GetSubmissions(eventId int) ([]*repository.Submission, error) {
	objectiveService := NewObjectiveService()
	objectives, err := objectiveService.GetObjectivesByEventId(eventId)
	if err != nil {
		return nil, err
	}
	return e.submissionRepository.GetSubmissionsForObjectives(objectives)
}

func (e *SubmissionService) SaveSubmission(submission *repository.Submission, submitter *repository.User) (*repository.Submission, error) {
	if submission.Id != 0 {
		existingSubmission, err := e.submissionRepository.GetSubmissionById(submission.Id)
		if err != nil {
			return nil, err
		}
		if existingSubmission.UserId != submitter.Id {
			return nil, fmt.Errorf("you are not allowed to edit this submission")
		}

		existingSubmission, err = e.submissionRepository.RemoveMatchFromSubmission(existingSubmission)
		if err != nil {
			return nil, err
		}
		existingSubmission.ObjectiveId = submission.ObjectiveId
		existingSubmission.Timestamp = submission.Timestamp
		existingSubmission.Number = submission.Number
		existingSubmission.Proof = submission.Proof
		existingSubmission.Comment = submission.Comment
		if existingSubmission.ApprovalStatus == repository.APPROVED {
			existingSubmission.ApprovalStatus = repository.PENDING
		}
		return e.submissionRepository.SaveSubmission(existingSubmission)
	}
	submission.ApprovalStatus = repository.PENDING
	submission.User = submitter
	submission.UserId = submitter.Id
	return e.submissionRepository.SaveSubmission(submission)
}

func (e *SubmissionService) ReviewSubmission(submissionId int, submissionReview *repository.Submission, reviewer *repository.User) (*repository.Submission, error) {
	submission, err := e.submissionRepository.GetSubmissionById(submissionId)

	if err != nil {
		return nil, err
	}
	if !utils.Contains(reviewer.Permissions, "admin") {
		return nil, fmt.Errorf("you are not allowed to review submissions")
	}
	if submissionReview.ApprovalStatus != repository.APPROVED {
		submission, err = e.submissionRepository.RemoveMatchFromSubmission(submission)
		if err != nil {
			return nil, err
		}
	}
	submission.ApprovalStatus = submissionReview.ApprovalStatus
	submission.ReviewComment = submissionReview.ReviewComment
	submission.ReviewerId = &reviewer.Id
	return e.submissionRepository.SaveSubmission(submission)
}

func (e *SubmissionService) DeleteSubmission(id int, user *repository.User) error {
	submission, err := e.submissionRepository.GetSubmissionById(id)
	if err != nil {
		return err
	}
	if submission.UserId != user.Id {
		return fmt.Errorf("you are not allowed to delete this submission")
	}
	return e.submissionRepository.DeleteSubmission(id)
}
