package orders

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type orderItem struct {
	ID int64 `json:"id"`

	LocationID      int64  `json:"location_id"`
	LocationName    string `json:"location_name"`
	LocationAddress string `json:"location_address"`

	StatusID   int64  `json:"status_id"`
	StatusName string `json:"status_name"`

	TotalSum  string    `json:"total_sum"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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

// GetOrders godoc
// @Summary Список заказов
// @Description Возвращает список заказов (админ), доступных пользователю по его локациям, с пагинацией.
// @Description
// @Description Поддерживает фильтры по location_id и status_id.
// @Description location_id — только из локаций пользователя, иначе FORBIDDEN.
// @Description status_id — должен существовать, иначе STATUS_NOT_FOUND.
// @Tags admin
// @Accept json
// @Produce json
// @Param location_id query int false "Фильтр по локации (только доступные пользователю)"
// @Param status_id query int false "Фильтр по статусу"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param limit query int false "Размер страницы (по умолчанию 10, максимум 100)"
// @Success 200 {object} getOrdersResp
// @Router /admin/orders [get]
func getOrders(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	// optional filters
	var (
		locationID int64
		hasLoc     bool

		statusID int64
		hasStat  bool
	)

	// location_id (optional)
	locStr := strings.TrimSpace(c.Query("location_id"))
	if locStr != "" {
		v, err := strconv.ParseInt(locStr, 10, 64)
		if err != nil || v <= 0 {
			response.Err(c, http.StatusBadRequest, "LOCATION_INVALID", "location_id must be positive integer")
			return
		}
		locationID = v
		hasLoc = true

		var access bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM user_locations
				WHERE user_id=$1 AND location_id=$2
			)`, userID, locationID).Scan(&access)
		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		if !access {
			response.Err(c, http.StatusForbidden, "FORBIDDEN", "no access to this location")
			return
		}
	}

	// status_id (optional)
	statStr := strings.TrimSpace(c.Query("status_id"))
	if statStr != "" {
		v, err := strconv.ParseInt(statStr, 10, 64)
		if err != nil || v <= 0 {
			response.Err(c, http.StatusBadRequest, "STATUS_INVALID", "status_id must be positive integer")
			return
		}
		statusID = v
		hasStat = true

		// status must exist
		var statusExists bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM order_statuses WHERE id=$1)`,
			statusID,
		).Scan(&statusExists); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		if !statusExists {
			response.Err(c, http.StatusNotFound, "STATUS_NOT_FOUND", "status not found")
			return
		}
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

	// -------- COUNT --------
	var total int64

	switch {
	case hasLoc && hasStat:
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM orders
			WHERE location_id=$1 AND status_id=$2`,
			locationID, statusID).Scan(&total); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

	case hasLoc && !hasStat:
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM orders
			WHERE location_id=$1`,
			locationID).Scan(&total); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

	case !hasLoc && hasStat:
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM orders
			WHERE status_id=$1
			  AND location_id IN (
				  SELECT location_id FROM user_locations WHERE user_id=$2
			  )`,
			statusID, userID).Scan(&total); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

	default: // !hasLoc && !hasStat
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM orders
			WHERE location_id IN (
				SELECT location_id FROM user_locations WHERE user_id=$1
			)`,
			userID).Scan(&total); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
	}

	// -------- ORDERS --------
	orders := make([]orderItem, 0, limit)

	if total > 0 {
		var (
			rows *sql.Rows
			err  error
		)

		base := `
			SELECT 
				o.id,
				l.id, l.name, l.address,
				s.id, s.name,
				o.total_sum,
				o.comment,
				o.created_at
			FROM orders o
			JOIN locations l ON l.id=o.location_id
			JOIN order_statuses s ON s.id=o.status_id
		`

		switch {
		case hasLoc && hasStat:
			rows, err = db.Query(base+`
				WHERE o.location_id=$1 AND o.status_id=$2
				ORDER BY o.created_at DESC,o.id DESC
				LIMIT $3 OFFSET $4`,
				locationID, statusID, limit, offset)

		case hasLoc && !hasStat:
			rows, err = db.Query(base+`
				WHERE o.location_id=$1
				ORDER BY o.created_at DESC,o.id DESC
				LIMIT $2 OFFSET $3`,
				locationID, limit, offset)

		case !hasLoc && hasStat:
			rows, err = db.Query(base+`
				WHERE o.status_id=$1
				  AND o.location_id IN (
					  SELECT location_id FROM user_locations WHERE user_id=$2
				  )
				ORDER BY o.created_at DESC,o.id DESC
				LIMIT $3 OFFSET $4`,
				statusID, userID, limit, offset)

		default: // !hasLoc && !hasStat
			rows, err = db.Query(base+`
				WHERE o.location_id IN (
					SELECT location_id FROM user_locations WHERE user_id=$1
				)
				ORDER BY o.created_at DESC,o.id DESC
				LIMIT $2 OFFSET $3`,
				userID, limit, offset)
		}

		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		defer rows.Close()

		for rows.Next() {
			var o orderItem
			var comment sql.NullString

			if err := rows.Scan(
				&o.ID,
				&o.LocationID,
				&o.LocationName,
				&o.LocationAddress,
				&o.StatusID,
				&o.StatusName,
				&o.TotalSum,
				&comment,
				&o.CreatedAt,
			); err != nil {
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
