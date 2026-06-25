package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// 1. Production Optimization: Set Release Mode
	gin.SetMode(gin.ReleaseMode)

	// 2. Database Initialization
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}

	r := gin.Default()

	// 3. Trust Proxy for Render/Cloud Platforms
	r.SetTrustedProxies([]string{"127.0.0.1"})

	// 4. Static & Templates
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// 5. Health Check Route (Perfect for UptimeRobot)
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	// 6. Navigation Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/explorer", func(c *gin.Context) { c.HTML(http.StatusOK, "explorer.html", nil) })
	r.GET("/docs", func(c *gin.Context) { c.HTML(http.StatusOK, "docs.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })
	r.GET("/core", func(c *gin.Context) { c.HTML(http.StatusOK, "core.html", nil) })
	r.GET("/markets", func(c *gin.Context) { c.HTML(http.StatusOK, "markets.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	// 7. Portal Routes
	r.GET("/portal/user", func(c *gin.Context) { c.HTML(http.StatusOK, "user_portal.html", nil) })
	r.GET("/portal/organization", func(c *gin.Context) { c.HTML(http.StatusOK, "organization_portal.html", nil) })
	r.GET("/portal/miner", func(c *gin.Context) { c.HTML(http.StatusOK, "miner_portal.html", nil) })

	// 8. Auth & API Routes
	r.POST("/auth/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/user"}) })
	r.POST("/api/transfer", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "success"}) })

	r.Run(":8085")
}
