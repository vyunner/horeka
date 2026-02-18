package locations

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/locations")
	g.Use(middleware.AuthRequired(jwtSecret))
	g.Use(middleware.RoleRequired(db, "admin"))

	g.POST("", func(c *gin.Context) { createLocation(c, db) })
	g.PUT("/:id", func(c *gin.Context) { updateLocation(c, db) })
	g.DELETE("/:id", func(c *gin.Context) { deleteLocation(c, db) })
}
