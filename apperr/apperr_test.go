package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errTeamNotFound = New(ErrNotFound, "team not found")

func TestStatus(t *testing.T) {
	assert.Equal(t, http.StatusNotFound, Status(errTeamNotFound))
	assert.Equal(t, http.StatusNotFound, Status(fmt.Errorf("loading team 3: %w", errTeamNotFound)))
	assert.Equal(t, http.StatusConflict, Status(New(ErrConflict, "taken")))
	assert.Equal(t, http.StatusInternalServerError, Status(errors.New("boom")))
	assert.Equal(t, http.StatusOK, Status(nil))
}

func TestErrorMessageAndIs(t *testing.T) {
	assert.Equal(t, "team not found", errTeamNotFound.Error())
	assert.True(t, errors.Is(errTeamNotFound, ErrNotFound))
	assert.False(t, errors.Is(errTeamNotFound, ErrConflict))
	assert.True(t, errors.Is(fmt.Errorf("x: %w", errTeamNotFound), errTeamNotFound))
}
