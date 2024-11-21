package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRoutes(r *gin.Engine, db *gorm.DB) {
	eventRoutes := GetEventRoutes(db)
	for _, route := range eventRoutes {
		r.Handle(route.Method, route.Path, route.HandlerFunc)
	}
}
