package main

import (
	dbpkg "horeka/internal/db"
	"horeka/internal/handlers/auth"
	"horeka/internal/handlers/locations"
	"horeka/internal/handlers/userlocations"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	_ = godotenv.Load()

	if len(os.Args) >= 2 && os.Args[1] == "migrate" {
		runMigrateCmd(os.Args[2:])
		return
	}

	port := getenv("APP_PORT", "8080")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required in .env")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required in .env")
	}

	conn, err := dbpkg.ConnectDB(dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close()

	r := gin.New()
	r.Use(gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	auth.RegisterRoutes(r, conn, jwtSecret)
	locations.RegisterRoutes(r, conn, jwtSecret)
	userlocations.RegisterRoutes(r, conn, jwtSecret)

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
