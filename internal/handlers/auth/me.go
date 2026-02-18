package auth

import (
	"database/sql"
	"horeka/internal/middleware"
	"horeka/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type meResponse struct {
	ID    int64  `json:"id"`
	Phone string `json:"phone"`
}

func me(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var phone string
	err := db.QueryRow(
		`SELECT phone FROM users WHERE id=$1`,
		userID,
	).Scan(&phone)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, meResponse{
		ID:    userID,
		Phone: phone,
	})
}
