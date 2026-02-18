package locations

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

func deleteLocation(c *gin.Context, db *sql.DB) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Err(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}

	res, err := db.Exec(`DELETE FROM locations WHERE id=$1`, id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	aff, _ := res.RowsAffected()
	if aff == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "locations not found")
		return
	}

	response.OK(c, nil)
}
