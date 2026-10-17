// Package apperr defines the error kinds shared by the repository, service and
// controller layers. Lower layers define concrete domain errors (for example
// repository.ErrTeamNotFound) that wrap one of the kinds below, and the
// controller picks the HTTP response from the kind with errors.Is / Status.
package apperr

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalid      = errors.New("invalid request")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	// ErrUpstream marks failures of a third party (PoE API, Discord, Twitch, ...).
	ErrUpstream = errors.New("upstream service failed")
)

// Error is a domain error with a user facing message that belongs to a kind.
type Error struct {
	kind error
	msg  string
}

// New creates a domain error. Declare these as package level sentinels and
// compare them with errors.Is.
func New(kind error, msg string) *Error {
	return &Error{kind: kind, msg: msg}
}

func (e *Error) Error() string { return e.msg }

// Unwrap exposes the kind so errors.Is(err, apperr.ErrNotFound) works.
func (e *Error) Unwrap() error { return e.kind }

// Status maps an error to an HTTP status. Anything without a known kind is a 500.
func Status(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUpstream):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// IsKnown reports whether err carries one of the kinds, i.e. whether its
// message is safe to show to the client.
func IsKnown(err error) bool {
	return Status(err) != http.StatusInternalServerError
}
