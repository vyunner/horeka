package orders

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type orderItem struct {
	ID         int64     `json:"id"`
	LocationID int64     `json:"location_id"`
	StatusID   int64     `json:"status_id"`
	TotalSum   string    `json:"total_sum"`
	Comment    *string   `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type getOrdersResp struct {
	Orders []orderItem `json:"orders"`
	Meta   struct {
		Page       int   `json:"page"`
		Limit      int   `json:"limit"`
		Total      int64 `json:"total"`
		TotalPages int   `json:"total_pages"`
	} `json:"meta"`
}

func getOrders(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

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
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&total); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	rows, err := db.Query(
		`SELECT id, location_id, status_id, total_sum, comment, created_at
		 FROM orders
		 WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer rows.Close()

	orders := make([]orderItem, 0, limit)

	for rows.Next() {
		var o orderItem
		var comment sql.NullString

		if err := rows.Scan(&o.ID, &o.LocationID, &o.StatusID, &o.TotalSum, &comment, &o.CreatedAt); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		if comment.Valid {
			o.Comment = &comment.String
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	totalPages := (int(total) + limit - 1) / limit

	var resp getOrdersResp
	resp.Orders = orders
	resp.Meta.Page = page
	resp.Meta.Limit = limit
	resp.Meta.Total = total
	resp.Meta.TotalPages = totalPages

	response.OK(c, resp)
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
