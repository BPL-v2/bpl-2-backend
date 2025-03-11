package controller

import (
	"bpl/auth"
	"bpl/repository"

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
		handlerfuncs = append(handlerfuncs, route.HandlerFunc)
		group.Handle(route.Method, route.Path, handlerfuncs...)
	}
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
