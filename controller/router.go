package controller

import (
	"bpl/auth"
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

type RouteInfo struct {
	Method        string
	Path          string
	HandlerFunc   gin.HandlerFunc
	Authenticated bool
	RequiredRoles []repository.Permission
}

func SetRoutes(r *gin.Engine, cache *persistence.InMemoryStore) {
	poeClient := client.NewPoEClient(10, false, 600)

	routes := make([]RouteInfo, 0)
	group := r.Group("/api")
	routes = append(routes, setupEventController()...)
	routes = append(routes, setupTeamController()...)
	routes = append(routes, setupConditionController()...)
	routes = append(routes, setupObjectiveController()...)
	routes = append(routes, setupOauthController()...)
	routes = append(routes, setupUserController()...)
	routes = append(routes, setupScoringPresetController()...)
	routes = append(routes, setupSignupController()...)
	routes = append(routes, setupSubmissionController()...)
	routes = append(routes, setupScoreController(poeClient)...)
	routes = append(routes, setupLadderController()...)
	routes = append(routes, setupTeamSuggestionController()...)
	routes = append(routes, setupCharacterController()...)
	routes = append(routes, setupStreamController(cache)...)
	routes = append(routes, setupRecurringJobsController(poeClient)...)
	routes = append(routes, setupGuildStashController(poeClient)...)
	routes = append(routes, setupActivityController()...)
	for _, route := range routes {
		handlerfuncs := make([]gin.HandlerFunc, 0)
		handlerfuncs = append(handlerfuncs, AuthenticationMiddleware())
		if len(route.RequiredRoles) > 0 {
			handlerfuncs = append(handlerfuncs, AuthorizationMiddleware(route.RequiredRoles))
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
	ev := event.(*repository.Event)
	roles, ok := c.Get("userRoles")
	if !ev.Public && (!ok || len(roles.([]repository.Permission)) == 0) {
		c.AbortWithStatus(404)
		return nil
	}
	return ev
}

func AuthenticationMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		userRoles := getUserRoles(r)
		r.Set("userRoles", userRoles)
	}
}

func AuthorizationMiddleware(requiredRoles []repository.Permission) gin.HandlerFunc {
	return func(r *gin.Context) {
		userRoles := getUserRoles(r)
		if len(requiredRoles) == 0 {
			r.Next()
			return
		}
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if requiredRole == userRole {
					r.Next()
					return
				}
			}
		}
		r.AbortWithStatus(403)
	}
}

func getUserRoles(r *gin.Context) (permissions []repository.Permission) {
	authHeader := r.Request.Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return permissions
	}
	token, err := auth.ParseToken(authHeader[7:])
	if err != nil {
		return permissions
	}
	claims := &auth.Claims{}
	if !token.Valid {
		return permissions
	}
	claims.FromJWTClaims(token.Claims)
	if err := claims.Valid(); err != nil {
		return permissions
	}
	return utils.Map(claims.Permissions, func(perm string) repository.Permission {
		return repository.Permission(perm)
	})
}
