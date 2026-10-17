package controller

import (
	"bpl/apperr"
	"bpl/repository"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"domain not found", repository.ErrTeamNotFound, 404, `{"error":"team not found"}`},
		{"wrapped domain error", fmt.Errorf("loading: %w", repository.ErrGroupFull), 409, `{"error":"loading: group is full"}`},
		{"invalid", repository.ErrNotInGroup, 400, `{"error":"not in a group"}`},
		{"upstream", fmt.Errorf("%w: poe down", apperr.ErrUpstream), 502, `{"error":"upstream service failed: poe down"}`},
		{"unknown hides details", errors.New("pq: connection refused"), 500, `{"error":"internal server error"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
			respondError(c, tc.err)
			assert.Equal(t, tc.status, w.Code)
			assert.JSONEq(t, tc.body, w.Body.String())
		})
	}
}

func TestRespondErrorOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	run := func(err error, o ...errorOverride) (int, string) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
		respondError(c, err, o...)
		return w.Code, w.Body.String()
	}
	override := withStatus(500, repository.ErrUserNotFound)

	code, body := run(repository.ErrUserNotFound, override)
	assert.Equal(t, 500, code)
	assert.JSONEq(t, `{"error":"internal server error"}`, body)

	// the default for the kind still applies to everything not overridden
	code, _ = run(repository.ErrCharacterNotFound, override)
	assert.Equal(t, 404, code)

	code, body = run(fmt.Errorf("wrapped: %w", repository.ErrEventNotFound), withStatus(400, repository.ErrEventNotFound))
	assert.Equal(t, 400, code)
	assert.JSONEq(t, `{"error":"wrapped: event not found"}`, body)
}
