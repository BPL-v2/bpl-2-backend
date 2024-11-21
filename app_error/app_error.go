package app_error

import "github.com/gin-gonic/gin"

type statusError struct {
	error
	status int
}

func (e statusError) Unwrap() error {
	return e.error
}

func (e statusError) HTTPStatus() int {
	return e.status
}

func WithHTTPStatus(c *gin.Context, err error, status int) {
	c.JSON(status, gin.H{"error": err.Error()})
}
