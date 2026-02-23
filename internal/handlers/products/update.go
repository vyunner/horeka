package products

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type productUpdateReq struct {
	Name    string   `json:"name"`
	Aliases []string `json:"product_aliases"`
}

// UpdateProduct godoc
// @Summary Обновление продукта
// @Description Обновляет название продукта и полностью заменяет список алиасов.
// @Description Доступ только для роли admin.
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "ID продукта"
// @Param input body productUpdateReq true "Данные продукта"
// @Success 200
// @Router /products/{id} [put]
func updateProduct(c *gin.Context, db *sql.DB) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	var req productUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Err(c, http.StatusBadRequest, "NAME_REQUIRED", "name required")
		return
	}

	aliases := normalizeAliases(req.Aliases)

	tx, err := db.Begin()
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE products SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}

	if _, err := tx.Exec(`DELETE FROM product_aliases WHERE product_id = $1`, id); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if err := insertAliases(tx, id, aliases); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if err := tx.Commit(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, gin.H{
		"id":              id,
		"name":            name,
		"product_aliases": aliases,
	})
}
