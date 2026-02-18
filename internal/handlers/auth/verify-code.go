package auth

import (
	"database/sql"
	"horeka/internal/auth"
	"horeka/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type verifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func verifyCode(c *gin.Context, db *sql.DB, jwtSecret string) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	phone, ok := validatePhone(req.Phone)
	code := strings.TrimSpace(req.Code)

	if !ok || code == "" {
		response.Err(c, http.StatusBadRequest, "PHONE_AND_CODE_REQUIRED", "phone and code required")
		return
	}

	var smsID int64
	err := db.QueryRow(
		`UPDATE sms_codes
		 SET used_at = now()
		 WHERE phone=$1 AND code=$2 AND used_at IS NULL
		 RETURNING id`,
		phone, code,
	).Scan(&smsID)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusUnauthorized, "INVALID_CODE", "invalid code")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	var userID int64
	err = db.QueryRow(`SELECT id FROM users WHERE phone=$1`, phone).Scan(&userID)

	if err == sql.ErrNoRows {
		err = db.QueryRow(
			`INSERT INTO users(phone) VALUES ($1) RETURNING id`,
			phone,
		).Scan(&userID)

		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
	} else if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	token, err := auth.MakeToken(userID, jwtSecret)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "TOKEN_ERROR", "token error")
		return
	}

	response.OK(c, tokenResponse{AccessToken: token})
}
