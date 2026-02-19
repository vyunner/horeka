package userlocations

import (
	"database/sql"
	"net/http"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type unbindUserFromLocationReq struct {
	UserID     int64 `json:"user_id"`
	LocationID int64 `json:"location_id"`
}

// unbindUserFromLocation godoc
// @Summary Отвязать пользователя от локации
// @Description Удаляет привязку пользователя к локации (запись из user_locations) по паре user_id + location_id.
// @Description
// @Description Если пользователя или локации не существует — возвращается 404.
// @Description Если привязки user_id + location_id нет — возвращается 404 (bind not found).
// @Description При успешном удалении возвращается 200 без тела.
// @Tags user_locations
// @Accept json
// @Produce json
// @Param input body unbindUserFromLocationReq true "user_id и location_id"
// @Success 200
// @Router /user-locations/unbind [post]
func unbindUserFromLocation(c *gin.Context, db *sql.DB) {
	var req unbindUserFromLocationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	if req.UserID <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_USER_ID", "invalid user_id")
		return
	}
	if req.LocationID <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_LOCATION_ID", "invalid location_id")
		return
	}

	// Проверяем пользователя
	var tmp int
	err := db.QueryRow(`SELECT 1 FROM users WHERE id=$1`, req.UserID).Scan(&tmp)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// Проверяем локацию
	err = db.QueryRow(`SELECT 1 FROM locations WHERE id=$1`, req.LocationID).Scan(&tmp)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "location not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// Удаляем привязку
	res, err := db.Exec(
		`DELETE FROM user_locations
		 WHERE user_id=$1 AND location_id=$2`,
		req.UserID, req.LocationID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	aff, err := res.RowsAffected()
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if aff == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "bind not found")
		return
	}

	response.OK(c, nil)
}
