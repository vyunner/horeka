package userlocations

import (
	"database/sql"
	"net/http"
	"time"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type myLocationItem struct {
	LocationID int64     `json:"location_id"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	RoleID     *int64    `json:"role_id"`
	RoleName   *string   `json:"role_name"`
	GrantedAt  time.Time `json:"granted_at"`
}

// listMyLocations godoc
// @Summary Получить список моих локаций
// @Description Возвращает все локации, к которым текущий пользователь привязан (таблица user_locations).
// @Description
// @Description В ответе для каждой локации возвращаются:
// @Description - location_id, name, address
// @Description - role_id и role_name (могут быть null, если роль не назначена)
// @Description - granted_at — дата/время, когда была выдана привязка к локации
// @Description
// @Description Список отсортирован по granted_at по убыванию (сначала самые последние привязки).
// @Tags user_locations
// @Produce json
// @Security BearerAuth
// @Success 200 {array} myLocationItem
// @Router /user-locations/me [get]
func listMyLocations(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	rows, err := db.Query(
		`SELECT ul.location_id,
		        l.name,
		        l.address,
		        ul.role_id,
		        r.name,
		        ul.granted_at
		 FROM user_locations ul
		 JOIN locations l ON l.id = ul.location_id
		 LEFT JOIN roles r ON r.id = ul.role_id
		 WHERE ul.user_id = $1
		 ORDER BY ul.granted_at DESC`,
		userID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer rows.Close()

	var locations []myLocationItem
	for rows.Next() {
		var it myLocationItem
		if err := rows.Scan(
			&it.LocationID,
			&it.Name,
			&it.Address,
			&it.RoleID,
			&it.RoleName,
			&it.GrantedAt,
		); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
		locations = append(locations, it)
	}
	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, locations)
}
