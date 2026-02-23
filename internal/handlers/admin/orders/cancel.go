package orders

import (
	"database/sql"
	"net/http"
	"strconv"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

const statusCancelled = 4

// CancelOrder godoc
// @Summary Отмена заказа
// @Description Отменяет заказ, если он принадлежит одной из локаций текущего пользователя.
// @Description
// @Description Доступ только для роли admin.
// @Description При успешной отмене статус заказа изменяется на "отменён".
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "ID заказа"
// @Success 200
// @Router /admin/orders/{id}/cancel [post]
func cancelOrder(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Err(c, http.StatusBadRequest, "ORDER_INVALID", "id must be positive integer")
		return
	}

	res, err := db.Exec(`
		UPDATE orders
		SET status_id = $3
		WHERE id = $1
		  AND location_id IN (
			  SELECT location_id FROM user_locations WHERE user_id = $2
		  )
	`, orderID, userID, statusCancelled)
	if err != nil {
		dbErr(c, err)
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		dbErr(c, err)
		return
	}
	if affected == 0 {
		response.Err(c, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		return
	}

	response.OK(c, nil)
}
