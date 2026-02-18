package userlocations

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/user-locations")
	g.Use(middleware.AuthRequired(jwtSecret))

	g.POST("",
		middleware.RoleRequired(db, "admin"),
		func(c *gin.Context) { bindUserToLocation(c, db) },
	)

	g.DELETE("",
		middleware.RoleRequired(db, "admin"),
		func(c *gin.Context) { unbindUserFromLocation(c, db) },
	)

	g.GET("/me", func(c *gin.Context) { listMyLocations(c, db) })
}
