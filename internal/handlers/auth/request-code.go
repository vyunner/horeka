package auth

import (
	"database/sql"
	"horeka/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type phoneRequest struct {
	Phone string `json:"phone"`
}

func requestCode(c *gin.Context, db *sql.DB) {
	var req phoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	phone, ok := validatePhone(req.Phone)
	if !ok {
		response.Err(c, http.StatusBadRequest, "INVALID_PHONE", "invalid phone")
		return
	}

	// удаляем старые коды
	if _, err := db.Exec(`DELETE FROM sms_codes WHERE phone=$1`, phone); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	code := "123456" // dev код

	if _, err := db.Exec(`INSERT INTO sms_codes(phone, code) VALUES ($1,$2)`, phone, code); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, nil)
}
