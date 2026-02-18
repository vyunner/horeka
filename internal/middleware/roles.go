package middleware

import (
	"database/sql"
	"horeka/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RoleRequired(db *sql.DB, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, ok := CurrentUserID(c)
		if !ok {
			response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
			c.Abort()
			return
		}

		query := `
			SELECT r.name
			FROM roles r
			JOIN user_roles ur ON ur.role_id = r.id
			WHERE ur.user_id = $1
		`

		rows, err := db.Query(query, userID)
		if err != nil {
			// временно можно вывести err в лог, чтобы видеть реальную причину
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "database error")
			c.Abort()
			return
		}
		defer rows.Close()

		userRoles := map[string]bool{}

		for rows.Next() {
			var r string
			if err := rows.Scan(&r); err != nil {
				response.Err(c, http.StatusInternalServerError, "DB_ERROR", "database error")
				c.Abort()
				return
			}
			userRoles[r] = true
		}
		if err := rows.Err(); err != nil {
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", "database error")
			c.Abort()
			return
		}

		for _, required := range roles {
			if userRoles[required] {
				c.Next()
				return
			}
		}

		response.Err(c, http.StatusForbidden, "FORBIDDEN", "insufficient role")
		c.Abort()
	}
}
