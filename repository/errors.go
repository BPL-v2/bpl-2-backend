package repository

import (
	"errors"

	"bpl/apperr"

	"gorm.io/gorm"
)

// Domain errors returned by repositories. Each wraps an apperr kind so that
// callers can either match the specific error or just the kind.
var (
	ErrTeamNotFound     = apperr.New(apperr.ErrNotFound, "team not found")
	ErrTeamIconNotFound = apperr.New(apperr.ErrNotFound, "team icon not found")
	ErrTeamUserNotFound = apperr.New(apperr.ErrNotFound, "user is not in a team")

	ErrEventNotFound             = apperr.New(apperr.ErrNotFound, "event not found")
	ErrUserNotFound              = apperr.New(apperr.ErrNotFound, "user not found")
	ErrSignupNotFound            = apperr.New(apperr.ErrNotFound, "signup not found")
	ErrObjectiveNotFound         = apperr.New(apperr.ErrNotFound, "objective not found")
	ErrScoringRuleNotFound       = apperr.New(apperr.ErrNotFound, "scoring rule not found")
	ErrSubmissionNotFound        = apperr.New(apperr.ErrNotFound, "submission not found")
	ErrCharacterNotFound         = apperr.New(apperr.ErrNotFound, "character not found")
	ErrPobNotFound               = apperr.New(apperr.ErrNotFound, "PoB export not found")
	ErrTeamSheetEntryNotFound    = apperr.New(apperr.ErrNotFound, "team sheet entry not found")
	ErrAchievementNotFound       = apperr.New(apperr.ErrNotFound, "achievement not found")
	ErrItemWishNotFound          = apperr.New(apperr.ErrNotFound, "item wish not found")
	ErrGuildStashTabNotFound     = apperr.New(apperr.ErrNotFound, "guild stash tab not found")
	ErrGuildNotFound             = apperr.New(apperr.ErrNotFound, "guild not found")
	ErrEngagementNotFound        = apperr.New(apperr.ErrNotFound, "engagement not found")
	ErrOauthNotFound             = apperr.New(apperr.ErrNotFound, "oauth account not found")
	ErrClientCredentialsNotFound = apperr.New(apperr.ErrNotFound, "client credentials not found")
	ErrCachedDataNotFound        = apperr.New(apperr.ErrNotFound, "cached data not found")
)

// translate converts low level database errors into the repository's domain
// errors. notFound is returned for gorm.ErrRecordNotFound; pass nil to fall back
// to a generic apperr.ErrNotFound. Unknown errors are returned unchanged.
func translate(err error, notFound error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		if notFound == nil {
			return apperr.ErrNotFound
		}
		return notFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return apperr.New(apperr.ErrConflict, "resource already exists")
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return apperr.New(apperr.ErrInvalid, "referenced resource does not exist")
	default:
		return err
	}
}
