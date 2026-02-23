package orders

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"horeka/internal/response"

	"github.com/gin-gonic/gin"
)

var qtyRegex = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

const statusAssembled = 2

type setProductItem struct {
	RequestID *int64 `json:"request_id"` // optional
	ProductID int64  `json:"product_id"` // required
	Quantity  string `json:"quantity"`   // required: "1.25" (max 2 decimals)
	Unit      string `json:"unit"`       // required
	Price     int    `json:"price"`      // required, >= 0
	PriceCost int    `json:"price_cost"` // required, >= 0
	Available *bool  `json:"available"`  // optional, default true
}

type setProductsReq struct {
	Items []setProductItem `json:"items"`
}

// SetOrderProducts godoc
// @Summary Установить товары в заказе
// @Description Полностью заменяет список товаров заказа на переданный в запросе.
// @Description
// @Description Операция выполняется в транзакции: сначала удаляются все текущие позиции, затем вставляются новые.
// @Description Если указан request_id — проверяется, что он принадлежит этому заказу.
// @Description После обновления пересчитываются total_sum и total_sum_cost.
// @Description Если текущий статус заказа "draft" — статус меняется на "assembled".
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "ID заказа"
// @Param input body setProductsReq true "Список товаров заказа"
// @Success 200
// @Router /admin/orders/{id}/products [put]
func setOrderProducts(c *gin.Context, db *sql.DB) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Err(c, http.StatusBadRequest, "ORDER_INVALID", "id must be positive integer")
		return
	}

	var req setProductsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	if len(req.Items) == 0 {
		response.Err(c, http.StatusBadRequest, "ITEMS_REQUIRED", "items required")
		return
	}

	if errH := validateSetItems(req.Items); errH != nil {
		c.JSON(http.StatusBadRequest, errH)
		return
	}

	tx, err := db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "TX_ERROR", "begin tx failed")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var statusID int
	err = tx.QueryRowContext(c.Request.Context(),
		`SELECT status_id FROM orders WHERE id=$1 FOR UPDATE`,
		orderID,
	).Scan(&statusID)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		return
	}
	if err != nil {
		dbErr(c, err)
		return
	}

	if _, err = tx.ExecContext(c.Request.Context(),
		`DELETE FROM order_products WHERE order_id=$1`, orderID,
	); err != nil {
		dbErr(c, err)
		return
	}

	insStmt, err := tx.PrepareContext(c.Request.Context(), `
		INSERT INTO order_products
			(order_id, request_id, product_id, quantity, unit, price, total_sum, available, price_cost, total_sum_cost)
		VALUES
			($1::bigint,
			 $2::bigint,
			 $3::bigint,
			 $4::numeric(10,2),
			 $5::text,
			 $6::int,
			 floor(($4::numeric(10,2)) * ($6::numeric))::int,
			 $7::boolean,
			 $8::int,
			 floor(($4::numeric(10,2)) * ($8::numeric))::int)
	`)
	if err != nil {
		dbErr(c, err)
		return
	}
	defer insStmt.Close()

	for i, it := range req.Items {
		if it.RequestID != nil {
			var exists bool
			if err := tx.QueryRowContext(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM order_requests WHERE id=$1 AND order_id=$2)`,
				*it.RequestID, orderID,
			).Scan(&exists); err != nil {
				dbErr(c, err)
				return
			}
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id", "index": i})
				return
			}
		}

		available := it.Available == nil || *it.Available

		if _, err = insStmt.ExecContext(c.Request.Context(),
			orderID,
			it.RequestID,
			it.ProductID,
			it.Quantity,
			it.Unit,
			it.Price,
			available,
			it.PriceCost,
		); err != nil {
			dbErr(c, err)
			return
		}
	}

	if _, err = tx.ExecContext(c.Request.Context(), `
		UPDATE orders o
		SET
			total_sum      = COALESCE((SELECT SUM(total_sum)      FROM order_products WHERE order_id = o.id), 0),
			total_sum_cost = COALESCE((SELECT SUM(total_sum_cost) FROM order_products WHERE order_id = o.id), 0)
		WHERE o.id = $1
	`, orderID); err != nil {
		dbErr(c, err)
		return
	}

	newStatus := statusID
	if statusID == statusDraft {
		if _, err = tx.ExecContext(c.Request.Context(),
			`UPDATE orders SET status_id=$1 WHERE id=$2`, statusAssembled, orderID,
		); err != nil {
			dbErr(c, err)
			return
		}
		newStatus = statusAssembled
	}

	if err := tx.Commit(); err != nil {
		dbErr(c, err)
		return
	}

	response.OK(c, gin.H{"order_id": orderID, "status_id": newStatus})
}

func validateSetItems(items []setProductItem) gin.H {
	for i, it := range items {
		switch {
		case it.ProductID <= 0:
			return gin.H{"error": "product_id required", "index": i}
		case !qtyRegex.MatchString(it.Quantity):
			return gin.H{"error": "quantity must be a number with max 2 decimals", "index": i}
		case it.Unit == "":
			return gin.H{"error": "unit required", "index": i}
		case it.Price < 0:
			return gin.H{"error": "price must be >= 0", "index": i}
		case it.PriceCost < 0:
			return gin.H{"error": "price_cost must be >= 0", "index": i}
		}
	}
	return nil
}
