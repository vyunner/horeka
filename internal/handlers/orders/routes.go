package orders

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/orders")
	g.Use(middleware.AuthRequired(jwtSecret))

	g.POST("", func(c *gin.Context) {
		createOrder(c, db)
	})

	g.GET("", func(c *gin.Context) {
		getOrders(c, db)
	})

	g.GET("/:id", func(c *gin.Context) {
		showOrder(c, db)
	})
}
