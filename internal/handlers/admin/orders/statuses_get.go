package orders

import (
	"database/sql"
	"net/http"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type orderStatusItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// оставляем struct-тип, но делаем его алиасом массива
type getOrderStatusesResp []orderStatusItem

// GetOrderStatuses godoc
// @Summary Список статусов заказов
// @Description Возвращает все доступные статусы заказов.
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} orderStatusItem
// @Router /admin/orders/statuses [get]
func getOrderStatuses(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, name
		FROM order_statuses
		ORDER BY id`)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer rows.Close()

	statuses := make([]orderStatusItem, 0, 16)

	for rows.Next() {
		var s orderStatusItem
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		statuses = append(statuses, s)
	}

	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, getOrderStatusesResp(statuses))
}
