package internal

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type createItemReq struct {
	Title string `json:"title"`
}

type updateItemReq struct {
	Title string `json:"title"`
}

func RegisterItemRoutes(r *gin.Engine, db *sql.DB, jwtSecret string) {
	g := r.Group("/items", AuthRequired(jwtSecret))
	{
		g.POST("", func(c *gin.Context) { createItem(c, db) })
		g.GET("", func(c *gin.Context) { listItems(c, db) })
		g.GET("/:id", func(c *gin.Context) { getItem(c, db) })
		g.PUT("/:id", func(c *gin.Context) { updateItem(c, db) })
		g.DELETE("/:id", func(c *gin.Context) { deleteItem(c, db) })
	}
}

func createItem(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createItemReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Title) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	var it Item
	it.UserID = userID
	it.Title = req.Title

	err := db.QueryRow(
		`INSERT INTO items (user_id, title) VALUES ($1, $2)
		 RETURNING id, created_at`,
		userID, req.Title,
	).Scan(&it.ID, &it.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusCreated, it)
}

func listItems(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rows, err := db.Query(
		`SELECT id, user_id, title, created_at
		 FROM items
		 WHERE user_id = $1
		 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.UserID, &it.Title, &it.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		items = append(items, it)
	}

	c.JSON(http.StatusOK, items)
}

func getItem(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var it Item
	err = db.QueryRow(
		`SELECT id, user_id, title, created_at
		 FROM items
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&it.ID, &it.UserID, &it.Title, &it.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, it)
}

func updateItem(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req updateItemReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Title) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	var it Item
	err = db.QueryRow(
		`UPDATE items
		 SET title = $1
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, user_id, title, created_at`,
		req.Title, id, userID,
	).Scan(&it.ID, &it.UserID, &it.Title, &it.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, it)
}

func deleteItem(c *gin.Context, db *sql.DB) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := db.Exec(`DELETE FROM items WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	aff, _ := res.RowsAffected()
	if aff == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
