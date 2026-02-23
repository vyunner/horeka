package products

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type productCreateReq struct {
	Name    string   `json:"name"`
	Aliases []string `json:"product_aliases"`
}

type productResp struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Aliases   []string `json:"product_aliases"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// CreateProduct godoc
// @Summary Создание продукта
// @Description Создает новый продукт с алиасами.
// @Description Доступ только для роли admin.
// @Tags products
// @Accept json
// @Produce json
// @Param input body productCreateReq true "Данные продукта"
// @Success 200 {object} productResp
// @Router /products [post]
func createProduct(c *gin.Context, db *sql.DB) {
	var req productCreateReq
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

	var id int64
	var createdAt time.Time

	err = tx.QueryRow(
		`INSERT INTO products(name)
		 VALUES ($1)
		 RETURNING id, created_at`,
		name,
	).Scan(&id, &createdAt)
	if err != nil {
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

	response.OK(c, productResp{
		ID:        id,
		Name:      name,
		Aliases:   aliases,
		CreatedAt: createdAt.Format(time.RFC3339),
	})
}
