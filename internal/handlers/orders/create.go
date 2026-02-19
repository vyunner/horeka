package orders

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

type createReq struct {
	LocationID int64        `json:"location_id"`
	Comment    string       `json:"comment"`
	Requests   []requestInp `json:"requests"`
}

type createResp struct {
	OrderID   int64         `json:"order_id"`
	StatusID  int64         `json:"status_id"`
	TotalSum  string        `json:"total_sum"`
	Comment   string        `json:"comment,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	Requests  []requestResp `json:"requests"`
}

type requestResp struct {
	ID          int64     `json:"id"`
	RawName     string    `json:"raw_name"`
	RawAmount   string    `json:"raw_amount"`
	IsAvailable *bool     `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
}

// createOrder godoc
// @Summary Создание нового заказа
// @Description Создает заказ для текущего авторизованного пользователя в указанной локации.
// @Description
// @Description В теле запроса передается список позиций (requests) в свободной форме: raw_name и raw_amount.
// @Description Сервер сохраняет эти позиции как заявки и создает заказ в начальном статусе "новый" (status_id = 1).
// @Description
// @Description comment — необязательный комментарий к заказу. Если передан пустым, в базе будет NULL и поле может отсутствовать в ответе.
// @Description
// @Description В ответе возвращается созданный заказ и список сохраненных заявок с их ID и временем создания.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body createReq true "Локация, комментарий (опц.), список заявок"
// @Success 200 {object} createResp
// @Router /orders [post]
func createOrder(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}

	if req.LocationID <= 0 {
		response.Err(c, http.StatusBadRequest, "LOCATION_REQUIRED", "location_id required")
		return
	}

	if len(req.Requests) == 0 {
		response.Err(c, http.StatusBadRequest, "REQUESTS_REQUIRED", "requests required")
		return
	}

	for i := range req.Requests {
		req.Requests[i].RawName = strings.TrimSpace(req.Requests[i].RawName)
		req.Requests[i].RawAmount = strings.TrimSpace(req.Requests[i].RawAmount)
		if req.Requests[i].RawName == "" || req.Requests[i].RawAmount == "" {
			response.Err(c, http.StatusBadRequest, "INVALID_REQUEST_ITEM", "each request must have raw_name and raw_amount")
			return
		}
	}

	comment := strings.TrimSpace(req.Comment)

	const initialStatusID int64 = 1

	tx, err := db.Begin()
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var resp createResp

	var commentNull sql.NullString
	if comment != "" {
		commentNull = sql.NullString{String: comment, Valid: true}
	}

	err = tx.QueryRow(
		`INSERT INTO orders(user_id, location_id, status_id, total_sum, comment)
		 VALUES ($1, $2, $3, 0, $4)
		 RETURNING id, status_id, total_sum, comment, created_at`,
		userID, req.LocationID, initialStatusID, commentNull,
	).Scan(&resp.OrderID, &resp.StatusID, &resp.TotalSum, &commentNull, &resp.CreatedAt)

	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	if commentNull.Valid {
		resp.Comment = commentNull.String
	}

	resp.Requests = make([]requestResp, 0, len(req.Requests))

	for _, r := range req.Requests {
		var rr requestResp

		err = tx.QueryRow(
			`INSERT INTO order_requests(order_id, raw_name, raw_amount)
			 VALUES ($1, $2, $3)
			 RETURNING id, raw_name, raw_amount, is_available, created_at`,
			resp.OrderID, r.RawName, r.RawAmount,
		).Scan(&rr.ID, &rr.RawName, &rr.RawAmount, &rr.IsAvailable, &rr.CreatedAt)

		if err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
			return
		}

		resp.Requests = append(resp.Requests, rr)
	}

	if err := tx.Commit(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
		return
	}

	response.OK(c, resp)
}
