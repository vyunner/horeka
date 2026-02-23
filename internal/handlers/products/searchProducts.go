package products

import (
	"database/sql"
	"net/http"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

// SearchProducts godoc
// @Summary Поиск продуктов
// @Description Ищет продукты по названию и алиасам.
// @Description
// @Description Параметр q обязателен. Поиск регистронезависимый, поддерживает подстроку и fuzzy (similarity).
// @Description Результаты ранжируются: сначала точное попадание подстрокой, затем по similarity.
// @Tags products
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param limit query int false "Размер страницы (по умолчанию 10, максимум 100)"
// @Success 200 {object} getProductsResp
// @Router /products/search [get]
func searchProducts(c *gin.Context, db *sql.DB) {
	qRaw := strings.TrimSpace(c.Query("q"))
	if qRaw == "" {
		response.Err(c, http.StatusBadRequest, "Q_REQUIRED", "q required")
		return
	}

	q := strings.ToLower(qRaw)

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// COUNT
	var total int64
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM products p
		WHERE
			lower(p.name) LIKE '%' || $1 || '%'
			OR similarity(lower(p.name), $1) >= 0.25
			OR EXISTS (
				SELECT 1
				FROM product_aliases a
				WHERE a.product_id = p.id
				  AND (
					lower(a.alias) LIKE '%' || $1 || '%'
					OR similarity(lower(a.alias), $1) >= 0.25
				  )
			)
	`, q).Scan(&total); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if total == 0 {
		var resp getProductsResp
		resp.Products = []productItem{}
		resp.Meta.Page = page
		resp.Meta.Limit = limit
		resp.Meta.Total = 0
		resp.Meta.TotalPages = 0
		response.OK(c, resp)
		return
	}

	// LIST (с ранжированием)
	rows, err := db.Query(`
		SELECT p.id, p.name, p.created_at
		FROM products p
		WHERE
			lower(p.name) LIKE '%' || $1 || '%'
			OR similarity(lower(p.name), $1) >= 0.25
			OR EXISTS (
				SELECT 1
				FROM product_aliases a
				WHERE a.product_id = p.id
				  AND (
					lower(a.alias) LIKE '%' || $1 || '%'
					OR similarity(lower(a.alias), $1) >= 0.25
				  )
			)
		ORDER BY
			-- сначала подстрока
			CASE
				WHEN lower(p.name) LIKE '%' || $1 || '%' THEN 2
				WHEN EXISTS (
					SELECT 1
					FROM product_aliases a
					WHERE a.product_id = p.id
					  AND lower(a.alias) LIKE '%' || $1 || '%'
				) THEN 1
				ELSE 0
			END DESC,

			-- потом fuzzy score
			GREATEST(
				similarity(lower(p.name), $1),
				COALESCE((
					SELECT MAX(similarity(lower(a.alias), $1))
					FROM product_aliases a
					WHERE a.product_id = p.id
				), 0)
			) DESC,

			-- стабильная сортировка
			p.created_at DESC,
			p.id DESC
		LIMIT $2 OFFSET $3
	`, q, limit, offset)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer rows.Close()

	products := make([]productItem, 0, limit)
	ids := make([]int64, 0, limit)

	for rows.Next() {
		var p productItem
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		p.Aliases = []string{}
		products = append(products, p)
		ids = append(ids, p.ID)
	}
	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	aliasMap, err := fetchAliasesByProductIDs(db, ids)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	for i := range products {
		if a, ok := aliasMap[products[i].ID]; ok {
			products[i].Aliases = a
		}
	}

	totalPages := (int(total) + limit - 1) / limit

	var resp getProductsResp
	resp.Products = products
	resp.Meta.Page = page
	resp.Meta.Limit = limit
	resp.Meta.Total = total
	resp.Meta.TotalPages = totalPages

	response.OK(c, resp)
}
