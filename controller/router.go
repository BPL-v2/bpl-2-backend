package controller

import (
	"bpl/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RouteInfo struct {
	Method        string
	Path          string
	HandlerFunc   gin.HandlerFunc
	Authenticated bool
	RoleRequired  []string
}

func SetRoutes(r *gin.Engine, db *gorm.DB) {
	routes := make([]RouteInfo, 0)
	routes = append(routes, setupEventController(db)...)
	routes = append(routes, setupTeamController(db)...)
	routes = append(routes, setupConditionController(db)...)
	routes = append(routes, setupScoringCategoryController(db)...)
	routes = append(routes, setupObjectiveController(db)...)
	routes = append(routes, setupOauthController(db)...)
	routes = append(routes, setupUserController(db)...)
	for _, route := range routes {
		handlerfuncs := make([]gin.HandlerFunc, 0)
		if route.Authenticated {
			handlerfuncs = append(handlerfuncs, AuthMiddleware(route.RoleRequired))
		}
		handlerfuncs = append(handlerfuncs, route.HandlerFunc)
		r.Handle(route.Method, route.Path, handlerfuncs...)
	}
}
func AuthMiddleware(roles []string) gin.HandlerFunc {
	return func(r *gin.Context) {
		authCookie, err := r.Cookie("auth")
		if err != nil {
			r.JSON(401, gin.H{"error": "Unauthenticated"})
			r.Abort()
			return
		}
		token, err := auth.ParseToken(authCookie)
		if err != nil {
			r.JSON(401, gin.H{"error": "Unauthenticated"})
			r.Abort()
			return
		}
		claims := &auth.Claims{}
		if !token.Valid {
			r.JSON(401, gin.H{"error": "Unauthenticated"})
			r.Abort()
			return
		}

		claims.FromJWTClaims(token.Claims)
		if err := claims.Valid(); err != nil {
			r.JSON(401, gin.H{"error": "Unauthenticated"})
			r.Abort()
			return
		}
		if len(roles) == 0 {
			r.Next()
			return
		}

		for _, requiredRole := range roles {
			for _, userRole := range claims.Permissions {
				if requiredRole == userRole {
					r.Next()
					return
				}
			}
		}
		r.JSON(403, gin.H{"error": "Unauthorized"})

	}
}
