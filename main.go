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
	// Initialize Database
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}

	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// Navigation Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/explorer", func(c *gin.Context) { c.HTML(http.StatusOK, "explorer.html", nil) })
	r.GET("/docs", func(c *gin.Context) { c.HTML(http.StatusOK, "docs.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })
	r.GET("/core", func(c *gin.Context) { c.HTML(http.StatusOK, "core.html", nil) })
	r.GET("/markets", func(c *gin.Context) { c.HTML(http.StatusOK, "markets.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	// Portal Routes
	r.GET("/portal/user", func(c *gin.Context) { c.HTML(http.StatusOK, "user_portal.html", nil) })
	r.GET("/portal/organization", func(c *gin.Context) { c.HTML(http.StatusOK, "organization_portal.html", nil) })
	r.GET("/portal/miner", func(c *gin.Context) { c.HTML(http.StatusOK, "miner_portal.html", nil) })

	// Auth & API
	r.POST("/auth/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/user"}) })
	r.POST("/api/transfer", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "success"}) })

	r.Run(":8085")
}
