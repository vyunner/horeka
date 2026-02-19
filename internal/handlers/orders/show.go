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

type requestItem struct {
	ID          int64     `json:"id"`
	RawName     string    `json:"raw_name"`
	RawAmount   string    `json:"raw_amount"`
	IsAvailable *bool     `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
}

type productItem struct {
	ID        int64     `json:"id"`
	RequestID *int64    `json:"request_id,omitempty"`
	ProductID int64     `json:"product_id"`
	Quantity  string    `json:"quantity"`
	Unit      string    `json:"unit"`
	Price     string    `json:"price"`
	TotalSum  string    `json:"total_sum"`
	Available bool      `json:"available"`
	CreatedAt time.Time `json:"created_at"`
}

type showOrderResp struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	LocationID int64     `json:"location_id"`
	StatusID   int64     `json:"status_id"`
	TotalSum   string    `json:"total_sum"`
	Comment    *string   `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`

	DetailsType string        `json:"details_type"` // "requests" | "products"
	Requests    []requestItem `json:"requests,omitempty"`
	Products    []productItem `json:"products,omitempty"`
}

// showOrder godoc
// @Summary Получить детали заказа по ID
// @Description Возвращает один заказ текущего авторизованного пользователя.
// @Description
// @Description Заказ ищется только среди заказов пользователя (id + user_id). Чужие заказы не отдаются.
// @Description Если заказ не найден — возвращается 404.
// @Description
// @Description Состав деталей зависит от статуса заказа:
// @Description - если status_id = 1 (новый) — возвращаются заявки из order_requests, а поле details_type = "requests"
// @Description - иначе — возвращаются товары из order_products, а поле details_type = "products"
// @Description
// @Description В ответе всегда присутствуют основные поля заказа (id, location_id, status_id, total_sum, comment, created_at)
// @Description и один из списков: requests или products.
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID заказа"
// @Success 200 {object} showOrderResp
// @Router /orders/{id} [get]
func showOrder(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	var resp showOrderResp
	resp.UserID = userID

	var comment sql.NullString
	err = db.QueryRow(
		`SELECT id, location_id, status_id, total_sum, comment, created_at
		 FROM orders
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&resp.ID, &resp.LocationID, &resp.StatusID, &resp.TotalSum, &comment, &resp.CreatedAt)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	if comment.Valid {
		resp.Comment = &comment.String
	}

	// status_id == 1 -> order_requests
	if resp.StatusID == 1 {
		resp.DetailsType = "requests"

		rows, err := db.Query(
			`SELECT id, raw_name, raw_amount, is_available, created_at
			 FROM order_requests
			 WHERE order_id = $1
			 ORDER BY id ASC`,
			resp.ID,
		)
		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		defer rows.Close()

		resp.Requests = make([]requestItem, 0)
		for rows.Next() {
			var r requestItem
			if err := rows.Scan(&r.ID, &r.RawName, &r.RawAmount, &r.IsAvailable, &r.CreatedAt); err != nil {
				response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
				return
			}
			resp.Requests = append(resp.Requests, r)
		}
		if err := rows.Err(); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

		response.OK(c, resp)
		return
	}

	// иначе -> order_products
	resp.DetailsType = "products"

	rows, err := db.Query(
		`SELECT id, request_id, product_id, quantity, unit, price, total_sum, available, created_at
		 FROM order_products
		 WHERE order_id = $1
		 ORDER BY id ASC`,
		resp.ID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer rows.Close()

	resp.Products = make([]productItem, 0)
	for rows.Next() {
		var p productItem
		var reqID sql.NullInt64

		if err := rows.Scan(&p.ID, &reqID, &p.ProductID, &p.Quantity, &p.Unit, &p.Price, &p.TotalSum, &p.Available, &p.CreatedAt); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

		if reqID.Valid {
			v := reqID.Int64
			p.RequestID = &v
		}

		resp.Products = append(resp.Products, p)
	}

	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
