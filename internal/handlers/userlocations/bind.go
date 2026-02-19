package userlocations

import (
	"database/sql"
	"net/http"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type bindUserToLocationReq struct {
	UserID     int64  `json:"user_id"`
	LocationID int64  `json:"location_id"`
	RoleID     *int64 `json:"role_id"` // null => без роли / стереть
}

// bindUserToLocation godoc
// @Summary Привязать пользователя к локации (назначить роль)
// @Description Создает или обновляет привязку пользователя к локации.
// @Description
// @Description Если привязки user_id + location_id еще нет — она будет создана.
// @Description Если привязка уже существует — будет обновлено только поле role_id.
// @Description
// @Description role_id — необязательное поле.
// @Description - role_id = null: привязка остается, роль будет очищена (role_id станет NULL)
// @Description - role_id > 0: роль будет установлена/заменена на указанную
// @Description
// @Description Метод проверяет существование пользователя, локации и (если передан) роли.
// @Tags user_locations
// @Accept json
// @Produce json
// @Param input body bindUserToLocationReq true "user_id, location_id, role_id (опц.)"
// @Success 200
// @Router /user-locations/bind [post]
func bindUserToLocation(c *gin.Context, db *sql.DB) {
	var req bindUserToLocationReq
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
	if req.RoleID != nil && *req.RoleID <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ROLE_ID", "invalid role_id")
		return
	}

	// user exists
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

	// location exists
	err = db.QueryRow(`SELECT 1 FROM locations WHERE id=$1`, req.LocationID).Scan(&tmp)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "location not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// role exists (optional)
	if req.RoleID != nil {
		err = db.QueryRow(`SELECT 1 FROM roles WHERE id=$1`, *req.RoleID).Scan(&tmp)
		if err == sql.ErrNoRows {
			response.Err(c, http.StatusBadRequest, "INVALID_ROLE_ID", "invalid role_id")
			return
		}
		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
	}

	_, err = db.Exec(
		`INSERT INTO user_locations(user_id, location_id, role_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, location_id)
		 DO UPDATE SET role_id = EXCLUDED.role_id`,
		req.UserID, req.LocationID, req.RoleID,
	)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, nil)
}
