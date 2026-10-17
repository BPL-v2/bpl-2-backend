package service

import (
	"bpl/apperr"
	"bpl/client"
	"fmt"
)

// Domain errors returned by services. Each wraps an apperr kind so the
// controller can pick the HTTP status without knowing about the service internals.
var (
	ErrInvalidAuthHeader = apperr.New(apperr.ErrUnauthorized, "authorization header is invalid")
	ErrInvalidToken      = apperr.New(apperr.ErrUnauthorized, "invalid or expired token")

	ErrLastProvider            = apperr.New(apperr.ErrConflict, "cannot remove last provider")
	ErrSystemAchievementEdit   = apperr.New(apperr.ErrForbidden, "cannot modify system achievements")
	ErrSystemAchievementDelete = apperr.New(apperr.ErrForbidden, "cannot delete system achievements")
	ErrSubmissionNotOwned      = apperr.New(apperr.ErrForbidden, "you are not allowed to edit this submission")

	ErrCharacterHasNoUser = apperr.New(apperr.ErrInvalid, "character has no associated user")
	ErrNoPoEToken         = apperr.New(apperr.ErrInvalid, "user has no valid PoE token")

	ErrUnknownOauthState = apperr.New(apperr.ErrInvalid, "oauth state is unknown")
	ErrUnknownProvider   = apperr.New(apperr.ErrInvalid, "provider not found")

	// ErrNoScoreChanges is a control signal for the scoring loop, not a failure.
	ErrNoScoreChanges = fmt.Errorf("no changes in scores")
)

// upstreamError wraps a failed third party call so it surfaces as a 502 with the
// provider's description instead of a generic 500.
func upstreamError(what string, e *client.ClientError) error {
	return fmt.Errorf("%w: %s: %d %s", apperr.ErrUpstream, what, e.StatusCode, e.Description)
}
