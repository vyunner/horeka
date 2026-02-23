package products

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

// DeleteProduct godoc
// @Summary Удаление продукта
// @Description Удаляет продукт по ID.
// @Description Доступ только для роли admin.
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "ID продукта"
// @Success 200
// @Router /products/{id} [delete]
func deleteProduct(c *gin.Context, db *sql.DB) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	res, err := db.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	aff, _ := res.RowsAffected()
	if aff == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}

	response.OK(c, gin.H{"ok": true})
}
