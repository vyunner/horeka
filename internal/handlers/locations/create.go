package locations

import (
	"database/sql"
	"net/http"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type locationCreateReq struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type locationCreateResp struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at"`
}

func createLocation(c *gin.Context, db *sql.DB) {
	var req locationCreateReq
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

	var resp locationCreateResp
	err := db.QueryRow(
		`INSERT INTO locations(name, address)
		 VALUES ($1, $2)
		 RETURNING id, name, address, created_at`,
		name, address,
	).Scan(&resp.ID, &resp.Name, &resp.Address, &resp.CreatedAt)

	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
