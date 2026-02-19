package orders

import (
	"database/sql"
	"net/http"
	"strconv"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type cancelResp struct {
	ID       int64 `json:"id"`
	StatusID int64 `json:"status_id"`
}

// cancelOrder godoc
// @Summary Отмена заказа
// @Description Отменяет заказ текущего пользователя по ID.
// @Description
// @Description Отменить можно только свой заказ и только в статусе "новый" (status_id = 1).
// @Description Если заказ уже в обработке/завершен/в другом статусе — отмена запрещена.
// @Description
// @Description При успешной отмене статус заказа меняется на "отменен" (status_id = 4) и это значение возвращается в ответе.
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID заказа"
// @Success 200 {object} cancelResp
// @Router /orders/{id}/cancel [post]
func cancelOrder(c *gin.Context, db *sql.DB) {
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

	const cancelledStatusID int64 = 4

	tx, err := db.Begin()
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var ownerID int64
	var currentStatus int64

	err = tx.QueryRow(
		`SELECT user_id, status_id
		 FROM orders
		 WHERE id = $1
		 FOR UPDATE`,
		orderID,
	).Scan(&ownerID, &currentStatus)

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

	const statusNew int64 = 1

	if currentStatus != statusNew {
		response.Err(c, http.StatusConflict, "ORDER_NOT_CANCELLABLE", "order is not cancellable")
		return
	}

	// если хочешь запретить повторную отмену:
	if currentStatus == cancelledStatusID {
		response.Err(c, http.StatusConflict, "ALREADY_CANCELLED", "order already cancelled")
		return
	}

	_, err = tx.Exec(
		`UPDATE orders
		 SET status_id = $1
		 WHERE id = $2`,
		cancelledStatusID, orderID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if err := tx.Commit(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, cancelResp{ID: orderID, StatusID: cancelledStatusID})
}
