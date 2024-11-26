package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRoutes(r *gin.Engine, db *gorm.DB) {
	routes := make([]gin.RouteInfo, 0)
	routes = append(routes, setupEventController(db)...)
	routes = append(routes, setupTeamController(db)...)
	routes = append(routes, setupConditionController(db)...)
	routes = append(routes, setupScoringCategoryController(db)...)
	routes = append(routes, setupObjectiveController(db)...)
	routes = append(routes, setupOauthController(db)...)
	for _, route := range routes {
		r.Handle(route.Method, route.Path, route.HandlerFunc)
	}
}
