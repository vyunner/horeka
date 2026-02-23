package products

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type productItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Aliases   []string  `json:"product_aliases"`
	CreatedAt time.Time `json:"created_at"`
}

type getProductsResp struct {
	Products []productItem `json:"products"`
	Meta     struct {
		Page       int   `json:"page"`
		Limit      int   `json:"limit"`
		Total      int64 `json:"total"`
		TotalPages int   `json:"total_pages"`
	} `json:"meta"`
}

// GetProducts godoc
// @Summary Список продуктов
// @Description Возвращает список продуктов с алиасами (админ), с пагинацией.
// @Tags products
// @Accept json
// @Produce json
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param limit query int false "Размер страницы (по умолчанию 10, максимум 100)"
// @Success 200 {object} getProductsResp
// @Router /products [get]
func getProducts(c *gin.Context, db *sql.DB) {
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

	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&total); err != nil {
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

	rows, err := db.Query(`
		SELECT id, name, created_at
		FROM products
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
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

func fetchAliasesByProductIDs(db *sql.DB, ids []int64) (map[int64][]string, error) {
	out := make(map[int64][]string)
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := db.Query(`
		SELECT product_id, alias
		FROM product_aliases
		WHERE product_id = ANY($1)
		ORDER BY id ASC
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pid int64
		var alias string
		if err := rows.Scan(&pid, &alias); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], alias)
	}
	return out, rows.Err()
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
