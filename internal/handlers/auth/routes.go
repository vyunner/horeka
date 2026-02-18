package auth

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/auth")

	g.POST("/request-code", func(c *gin.Context) {
		requestCode(c, db)
	})

	g.POST("/verify-code", func(c *gin.Context) {
		verifyCode(c, db, jwtSecret)
	})

	g.GET("/me",
		middleware.AuthRequired(jwtSecret),
		func(c *gin.Context) {
			me(c, db)
		},
	)
}
