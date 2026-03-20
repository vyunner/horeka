package orders

import (
	"database/sql"

	"horeka/internal/middleware"
	"horeka/internal/telegram"

	"github.com/gin-gonic/gin"
)

// tgBot is set via SetBot from main.
var tgBot *telegram.Bot

// SetBot sets the Telegram bot for order notifications.
func SetBot(b *telegram.Bot) {
	tgBot = b
}

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

	g.PUT("/:id", func(c *gin.Context) {
		updateOrder(c, db)
	})

	g.POST("/:id/cancel", func(c *gin.Context) {
		cancelOrder(c, db)
	})
}
