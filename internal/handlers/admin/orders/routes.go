package orders

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/admin/orders")
	g.Use(middleware.AuthRequired(jwtSecret))
	g.Use(middleware.RoleRequired(db, "admin"))

	g.GET("", func(c *gin.Context) {
		getOrders(c, db)
	})

	g.GET("/statuses", func(c *gin.Context) {
		getOrderStatuses(c, db)
	})

	g.GET("/:id", func(c *gin.Context) {
		showOrder(c, db)
	})

	g.PUT("/:id/products", func(c *gin.Context) {
		setOrderProducts(c, db)
	})

	g.POST("/:id/cancel", func(c *gin.Context) {
		cancelOrder(c, db)
	})
}
