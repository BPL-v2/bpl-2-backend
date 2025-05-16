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

func (e *SubmissionService) SaveBulkSubmissions(submissions []*repository.Submission) ([]*repository.Submission, error) {
	persisted := make([]*repository.Submission, 0)
	for _, submission := range submissions {
		s, err := e.submissionRepository.SaveSubmission(submission)
		if err != nil {
			return nil, err
		}
		persisted = append(persisted, s)
	}
	return persisted, nil
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
		err = e.submissionRepository.RemoveMatchFromSubmission(existingSubmission)
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
	if submissionReview.ApprovalStatus == repository.APPROVED {
		err = e.submissionRepository.AddMatchToSubmission(submission)
	} else {
		err = e.submissionRepository.RemoveMatchFromSubmission(submission)
	}
	if err != nil {
		return nil, err
	}
	submission.ApprovalStatus = submissionReview.ApprovalStatus
	submission.ReviewComment = submissionReview.ReviewComment
	submission.ReviewerId = &reviewer.Id
	return e.submissionRepository.SaveSubmission(submission)
}

func (e *SubmissionService) GetSubmissionById(id int) (*repository.Submission, error) {
	submission, err := e.submissionRepository.GetSubmissionById(id)
	if err != nil {
		return nil, err
	}
	return submission, nil
}

func (e *SubmissionService) DeleteSubmission(submission *repository.Submission, user *repository.User) error {
	return e.submissionRepository.DeleteSubmission(submission.Id)
}
