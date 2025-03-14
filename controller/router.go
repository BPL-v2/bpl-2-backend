package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RouteInfo struct {
	Method        string
	Path          string
	HandlerFunc   gin.HandlerFunc
	Authenticated bool
	RequiredRoles []repository.Permission
}

func SetRoutes(r *gin.Engine) {
	routes := make([]RouteInfo, 0)
	group := r.Group("/api")
	routes = append(routes, setupEventController()...)
	routes = append(routes, setupTeamController()...)
	routes = append(routes, setupConditionController()...)
	routes = append(routes, setupScoringCategoryController()...)
	routes = append(routes, setupObjectiveController()...)
	routes = append(routes, setupOauthController()...)
	routes = append(routes, setupUserController()...)
	routes = append(routes, setupScoringPresetController()...)
	routes = append(routes, setupSignupController()...)
	routes = append(routes, setupSubmissionController()...)
	routes = append(routes, setupScoreController()...)
	routes = append(routes, setupStreamController()...)
	routes = append(routes, setupRecurringJobsController()...)
	routes = append(routes, setupLadderController()...)
	for _, route := range routes {
		handlerfuncs := make([]gin.HandlerFunc, 0)
		if route.Authenticated {
			handlerfuncs = append(handlerfuncs, AuthMiddleware(route.RequiredRoles))
		}
		handlerfuncs = append(handlerfuncs, LoadEventMiddleware())
		handlerfuncs = append(handlerfuncs, route.HandlerFunc)
		group.Handle(route.Method, route.Path, handlerfuncs...)
	}
}

func LoadEventMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		eventParam := r.Param("event_id")
		if eventParam == "" {
			r.Next()
			return
		}
		eventService := service.NewEventService()
		if eventParam == "current" {
			event, err := eventService.GetCurrentEvent()
			if err != nil {
				r.AbortWithStatus(404)
				return
			}
			r.Set("event", event)
			r.Next()
			return
		}
		eventId, err := strconv.Atoi(eventParam)
		if err != nil {
			r.AbortWithStatus(400)
			return
		}
		event, err := eventService.GetEventById(eventId)
		if err != nil {
			r.AbortWithStatus(404)
			return
		}
		r.Set("event", event)
		r.Next()
	}
}

func getEvent(c *gin.Context) *repository.Event {
	event, ok := c.Get("event")
	if !ok {
		c.AbortWithStatus(400)
		return nil
	}
	return event.(*repository.Event)
}

func AuthMiddleware(roles []repository.Permission) gin.HandlerFunc {
	return func(r *gin.Context) {
		authCookie, err := r.Cookie("auth")
		if err != nil {
			r.AbortWithStatus(401)
			return
		}
		token, err := auth.ParseToken(authCookie)
		if err != nil {
			r.AbortWithStatus(401)
			return
		}
		claims := &auth.Claims{}
		if !token.Valid {
			r.AbortWithStatus(401)
			return
		}
		claims.FromJWTClaims(token.Claims)
		if err := claims.Valid(); err != nil {
			r.AbortWithStatus(401)
			return
		}
		if len(roles) == 0 {
			r.Next()
			return
		}

		for _, requiredRole := range roles {
			for _, userRole := range claims.Permissions {
				if requiredRole == repository.Permission(userRole) {
					r.Next()
					return
				}
			}
		}
		r.AbortWithStatus(403)
	}
}
