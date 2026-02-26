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

	// Всегда присутствуют, даже если пустые: []
	OrderRequests []requestItem `json:"order_requests"`
	OrderProducts []productItem `json:"order_products"`
}

// showOrder godoc
// @Summary Получить детали заказа по ID
// @Description Возвращает один заказ текущего пользователя по ID вместе с деталями.
// @Description
// @Description В ответе всегда возвращаются оба массива: order_requests и order_products.
// @Description Если данных нет — возвращаются пустые массивы [].
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

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	resp := showOrderResp{
		UserID:        userID,
		OrderRequests: make([]requestItem, 0),
		OrderProducts: make([]productItem, 0),
	}

	var comment sql.NullString
	err = db.QueryRow(
		`SELECT id, location_id, status_id, total_sum, comment, created_at
		 FROM orders
		 WHERE id = $1 AND user_id = $2`,
		orderID, userID,
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

	// ---- order_requests (всегда) ----
	reqRows, err := db.Query(
		`SELECT id, raw_name, raw_amount, is_available, created_at
		 FROM order_requests
		 WHERE order_id = $1
		 ORDER BY id`,
		resp.ID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer reqRows.Close()

	for reqRows.Next() {
		var r requestItem
		if err := reqRows.Scan(&r.ID, &r.RawName, &r.RawAmount, &r.IsAvailable, &r.CreatedAt); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		resp.OrderRequests = append(resp.OrderRequests, r)
	}
	if err := reqRows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// ---- order_products (всегда) ----
	prodRows, err := db.Query(
		`SELECT id, request_id, product_id, quantity, unit, price, total_sum, available, created_at
		 FROM order_products
		 WHERE order_id = $1
		 ORDER BY id`,
		resp.ID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var p productItem
		var reqID sql.NullInt64

		if err := prodRows.Scan(&p.ID, &reqID, &p.ProductID, &p.Quantity, &p.Unit, &p.Price, &p.TotalSum, &p.Available, &p.CreatedAt); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

		if reqID.Valid {
			v := reqID.Int64
			p.RequestID = &v
		}

		resp.OrderProducts = append(resp.OrderProducts, p)
	}
	if err := prodRows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
