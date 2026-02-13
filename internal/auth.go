package internal

import (
	"database/sql"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

type phoneReq struct {
	Phone string `json:"phone"`
}

type verifyReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
}

type meResp struct {
	ID    int64  `json:"id"`
	Phone string `json:"phone"`
}

func RegisterAuthRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/auth")

	g.POST("/request-code", func(c *gin.Context) { requestCode(c, db) })
	g.POST("/verify-code", func(c *gin.Context) { verifyCode(c, db, jwtSecret) })
	g.GET("/me", AuthRequired(jwtSecret), func(c *gin.Context) { me(c, db) })
}

func requestCode(c *gin.Context, db *sql.DB) {
	var req phoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	phone, ok := validatePhone(req.Phone)
	if !ok {
		Err(c, http.StatusBadRequest, "INVALID_PHONE", "invalid phone")
		return
	}

	// чистим только использованные коды этого телефона
	if _, err := db.Exec(`DELETE FROM sms_codes WHERE phone=$1 AND used_at IS NOT NULL`, phone); err != nil {
		Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// DEV код
	code := "123456"

	_, err := db.Exec(`INSERT INTO sms_codes(phone, code) VALUES ($1,$2)`, phone, code)
	if err != nil {
		Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	OK(c, nil)
}

func verifyCode(c *gin.Context, db *sql.DB, jwtSecret string) {
	var req verifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	phone, ok := validatePhone(req.Phone)
	code := strings.TrimSpace(req.Code)
	if !ok || code == "" {
		Err(c, http.StatusBadRequest, "PHONE_AND_CODE_REQUIRED", "phone and code required")
		return
	}

	// одноразовый код: если найден и еще не использован — помечаем used_at
	var smsCodeID int64
	err := db.QueryRow(
		`UPDATE sms_codes
		 SET used_at = now()
		 WHERE phone=$1 AND code=$2 AND used_at IS NULL
		 RETURNING id`,
		phone, code,
	).Scan(&smsCodeID)

	if err == sql.ErrNoRows {
		Err(c, http.StatusUnauthorized, "INVALID_CODE", "invalid code")
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	// Создать пользователя, если нет
	var userID int64
	err = db.QueryRow(`SELECT id FROM users WHERE phone=$1`, phone).Scan(&userID)
	if err == sql.ErrNoRows {
		err = db.QueryRow(`INSERT INTO users(phone) VALUES ($1) RETURNING id`, phone).Scan(&userID)
		if err != nil {
			Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}
	} else if err != nil {
		Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	token, err := MakeToken(userID, jwtSecret)
	if err != nil {
		Err(c, http.StatusInternalServerError, "TOKEN_ERROR", "token error")
		return
	}

	OK(c, tokenResp{AccessToken: token})
}

func me(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var phone string
	err := db.QueryRow(`SELECT phone FROM users WHERE id=$1`, userID).Scan(&phone)
	if err == sql.ErrNoRows {
		Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	OK(c, meResp{
		ID:    userID,
		Phone: phone,
	})
}

func validatePhone(s string) (string, bool) {
	s = strings.TrimSpace(s)

	if len(s) != 11 {
		return "", false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	if s[0] != '7' {
		return "", false
	}
	return s, true
}
