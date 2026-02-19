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

type updateReq struct {
	LocationID int64        `json:"location_id"`
	Comment    string       `json:"comment"`
	Requests   []requestInp `json:"requests"`
}

type requestDTO struct {
	ID          int64     `json:"id"`
	RawName     string    `json:"raw_name"`
	RawAmount   string    `json:"raw_amount"`
	IsAvailable *bool     `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
}

type updateResp struct {
	ID         int64        `json:"id"`
	UserID     int64        `json:"user_id"`
	LocationID int64        `json:"location_id"`
	StatusID   int64        `json:"status_id"`
	TotalSum   string       `json:"total_sum"`
	Comment    *string      `json:"comment,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	Requests   []requestDTO `json:"requests"`
}

func updateOrder(c *gin.Context, db *sql.DB) {
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

	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	if req.LocationID <= 0 {
		response.Err(c, http.StatusBadRequest, "LOCATION_REQUIRED", "location_id required")
		return
	}
	if len(req.Requests) == 0 {
		response.Err(c, http.StatusBadRequest, "REQUESTS_REQUIRED", "requests required")
		return
	}

	for i := range req.Requests {
		req.Requests[i].RawName = strings.TrimSpace(req.Requests[i].RawName)
		req.Requests[i].RawAmount = strings.TrimSpace(req.Requests[i].RawAmount)
		if req.Requests[i].RawName == "" || req.Requests[i].RawAmount == "" {
			response.Err(c, http.StatusBadRequest, "INVALID_REQUEST_ITEM", "each request must have raw_name and raw_amount")
			return
		}
	}

	comment := strings.TrimSpace(req.Comment)
	var commentNull sql.NullString
	if comment != "" {
		commentNull = sql.NullString{String: comment, Valid: true}
	}

	tx, err := db.Begin()
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer func() { _ = tx.Rollback() }()

	// проверяем заказ + блокируем, чтобы параллельно никто не обновил статус
	var ownerID int64
	var statusID int64

	err = tx.QueryRow(
		`SELECT user_id, status_id
	 FROM orders
	 WHERE id = $1
	 FOR UPDATE`,
		orderID,
	).Scan(&ownerID, &statusID)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if ownerID != userID {
		response.Err(c, http.StatusForbidden, "FORBIDDEN", "not your order")
		return
	}

	// редактировать можно только "новый" (status_id = 1)
	if statusID != 1 {
		response.Err(c, http.StatusConflict, "ORDER_NOT_EDITABLE", "order is not editable")
		return
	}

	// обновляем заказ
	_, err = tx.Exec(
		`UPDATE orders
		 SET location_id = $1, comment = $2
		 WHERE id = $3 AND user_id = $4`,
		req.LocationID, commentNull, orderID, userID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// удаляем старые заявки
	_, err = tx.Exec(`DELETE FROM order_requests WHERE order_id = $1`, orderID)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// вставляем новые заявки
	newRequests := make([]requestDTO, 0, len(req.Requests))
	for _, r := range req.Requests {
		var rr requestDTO
		err = tx.QueryRow(
			`INSERT INTO order_requests(order_id, raw_name, raw_amount)
			 VALUES ($1, $2, $3)
			 RETURNING id, raw_name, raw_amount, is_available, created_at`,
			orderID, r.RawName, r.RawAmount,
		).Scan(&rr.ID, &rr.RawName, &rr.RawAmount, &rr.IsAvailable, &rr.CreatedAt)

		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		newRequests = append(newRequests, rr)
	}

	// читаем заказ для ответа
	var resp updateResp
	resp.UserID = userID
	resp.Requests = newRequests

	var commentOut sql.NullString
	err = tx.QueryRow(
		`SELECT id, location_id, status_id, total_sum, comment, created_at
		 FROM orders
		 WHERE id = $1 AND user_id = $2`,
		orderID, userID,
	).Scan(&resp.ID, &resp.LocationID, &resp.StatusID, &resp.TotalSum, &commentOut, &resp.CreatedAt)

	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	if commentOut.Valid {
		resp.Comment = &commentOut.String
	}

	if err := tx.Commit(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
