package locations

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type locationUpdateReq struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type locationUpdateResp struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at"`
}

// UpdateLocation godoc
// @Summary Обновление локации
// @Description Обновляет название и адрес локации по ID.
// @Description Доступ только для роли admin.
// @Tags locations
// @Accept json
// @Produce json
// @Param id path int true "ID локации"
// @Param input body locationUpdateReq true "Данные локации"
// @Success 200 {object} locationUpdateResp
// @Router /locations/{id} [put]
func updateLocation(c *gin.Context, db *sql.DB) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	var req locationUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	if name == "" || address == "" {
		response.Err(c, http.StatusBadRequest, "NAME_AND_ADDRESS_REQUIRED", "name and address required")
		return
	}

	var resp locationUpdateResp
	err = db.QueryRow(
		`UPDATE locations
		 SET name=$2, address=$3
		 WHERE id=$1
		 RETURNING id, name, address, created_at`,
		id, name, address,
	).Scan(&resp.ID, &resp.Name, &resp.Address, &resp.CreatedAt)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "locations not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
