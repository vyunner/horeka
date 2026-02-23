package orders

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"horeka/internal/middleware"
	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

const (
	statusDraft          = 1
	defaultOrderItemsCap = 32
)

type orderRequestItem struct {
	ID          int64     `json:"id"`
	OrderID     int64     `json:"order_id"`
	RawName     string    `json:"raw_name"`
	RawAmount   string    `json:"raw_amount"`
	IsAvailable *bool     `json:"is_available,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type orderProductItem struct {
	ID           int64     `json:"id"`
	OrderID      int64     `json:"order_id"`
	RequestID    *int64    `json:"request_id,omitempty"`
	ProductID    int64     `json:"product_id"`
	ProductName  string    `json:"product_name"`
	Quantity     string    `json:"quantity"`
	Unit         string    `json:"unit"`
	Price        int       `json:"price"`
	TotalSum     int       `json:"total_sum"`
	Available    bool      `json:"available"`
	CreatedAt    time.Time `json:"created_at"`
	PriceCost    *int      `json:"price_cost,omitempty"`
	TotalSumCost *int      `json:"total_sum_cost,omitempty"`
}

type showOrderResp struct {
	Order    orderItem          `json:"order"`
	Requests []orderRequestItem `json:"order_requests"` // всегда присутствует
	Products []orderProductItem `json:"order_products"` // nil если статус = черновик
}

func nullInt64ToIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func dbErr(c *gin.Context, err error) {
	log.Printf("db error: %v", err)
	response.Err(c, http.StatusInternalServerError, "DB_ERROR", "db error")
}

// ShowOrder godoc
// @Summary Детали заказа
// @Description Возвращает заказ по id, если он относится к одной из локаций текущего пользователя.
// @Description
// @Description Всегда возвращает список order_requests.
// @Description Список order_products возвращается только если статус заказа не "draft".
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "ID заказа"
// @Success 200 {object} showOrderResp
// @Router /admin/orders/{id} [get]
func showOrder(c *gin.Context, db *sql.DB) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok || userID <= 0 {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Err(c, http.StatusBadRequest, "ORDER_INVALID", "id must be positive integer")
		return
	}

	var o orderItem
	var comment sql.NullString

	err = db.QueryRow(`
		SELECT
			o.id,
			l.id, l.name, l.address,
			s.id, s.name,
			o.total_sum,
			o.comment,
			o.created_at
		FROM orders o
		JOIN locations l ON l.id = o.location_id
		JOIN order_statuses s ON s.id = o.status_id
		WHERE o.id = $1
		  AND o.location_id IN (
			  SELECT location_id FROM user_locations WHERE user_id = $2
		  )
	`, orderID, userID).Scan(
		&o.ID,
		&o.LocationID,
		&o.LocationName,
		&o.LocationAddress,
		&o.StatusID,
		&o.StatusName,
		&o.TotalSum,
		&comment,
		&o.CreatedAt,
	)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		return
	}
	if err != nil {
		dbErr(c, err)
		return
	}
	if comment.Valid {
		o.Comment = &comment.String
	}

	resp := showOrderResp{Order: o}

	// Requests грузим всегда: при черновике — это основные данные,
	// при остальных статусах — нужны для отображения raw_name/raw_amount в таблице продуктов.
	if err := fillRequests(c, db, orderID, &resp); err != nil {
		return
	}

	if o.StatusID != statusDraft {
		if err := fillProducts(c, db, orderID, &resp); err != nil {
			return
		}
	}

	response.OK(c, resp)
}

func fillRequests(c *gin.Context, db *sql.DB, orderID int64, resp *showOrderResp) error {
	rows, err := db.Query(`
		SELECT id, order_id, raw_name, raw_amount, is_available, created_at
		FROM order_requests
		WHERE order_id = $1
		ORDER BY created_at ASC, id ASC
	`, orderID)
	if err != nil {
		dbErr(c, err)
		return err
	}
	defer rows.Close()

	requests := make([]orderRequestItem, 0, defaultOrderItemsCap)
	for rows.Next() {
		var r orderRequestItem
		var isAvail sql.NullBool

		if err := rows.Scan(&r.ID, &r.OrderID, &r.RawName, &r.RawAmount, &isAvail, &r.CreatedAt); err != nil {
			dbErr(c, err)
			return err
		}
		if isAvail.Valid {
			v := isAvail.Bool
			r.IsAvailable = &v
		}
		requests = append(requests, r)
	}
	if err := rows.Err(); err != nil {
		dbErr(c, err)
		return err
	}

	resp.Requests = requests
	return nil
}

func fillProducts(c *gin.Context, db *sql.DB, orderID int64, resp *showOrderResp) error {
	rows, err := db.Query(`
		SELECT
			op.id,
			op.order_id,
			op.request_id,
			op.product_id,
			pr.name AS product_name,
			op.quantity::text,
			op.unit,
			op.price,
			op.total_sum,
			op.available,
			op.created_at,
			op.price_cost,
			op.total_sum_cost
		FROM order_products op
		JOIN products pr ON pr.id = op.product_id
		WHERE op.order_id = $1
		ORDER BY op.created_at ASC, op.id ASC
	`, orderID)
	if err != nil {
		dbErr(c, err)
		return err
	}
	defer rows.Close()

	products := make([]orderProductItem, 0, defaultOrderItemsCap)
	for rows.Next() {
		var p orderProductItem
		var reqID, pc, tsc sql.NullInt64

		if err := rows.Scan(
			&p.ID,
			&p.OrderID,
			&reqID,
			&p.ProductID,
			&p.ProductName,
			&p.Quantity,
			&p.Unit,
			&p.Price,
			&p.TotalSum,
			&p.Available,
			&p.CreatedAt,
			&pc,
			&tsc,
		); err != nil {
			dbErr(c, err)
			return err
		}

		p.RequestID = nullInt64ToPtr(reqID)
		p.PriceCost = nullInt64ToIntPtr(pc)
		p.TotalSumCost = nullInt64ToIntPtr(tsc)

		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		dbErr(c, err)
		return err
	}

	resp.Products = products
	return nil
}
