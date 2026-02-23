package products

import (
	"database/sql"

	"horeka/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {

	// ---- общая группа (только авторизация) ----
	base := r.Group("/products")
	base.Use(middleware.AuthRequired(jwtSecret))

	// search доступен всем авторизованным
	base.GET("/search", func(c *gin.Context) { searchProducts(c, db) })
	base.GET("", func(c *gin.Context) { getProducts(c, db) })

	// ---- admin группа ----
	admin := base.Group("")
	admin.Use(middleware.RoleRequired(db, "admin"))

	admin.POST("", func(c *gin.Context) { createProduct(c, db) })
	admin.PUT("/:id", func(c *gin.Context) { updateProduct(c, db) })
	admin.DELETE("/:id", func(c *gin.Context) { deleteProduct(c, db) })
}
