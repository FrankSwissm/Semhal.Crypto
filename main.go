package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Models ---
type Account struct {
	Address string  `gorm:"primaryKey" json:"address"`
	Balance float64 `gorm:"default:0.0" json:"balance"`
	IsOrg   bool    `gorm:"default:false" json:"is_org"`
}

var db *gorm.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	db.AutoMigrate(&Account{})

	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// --- Routes ---
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"total_supply": "1,250,000", "total_nodes": "48"})
	})

	r.POST("/auth/login", loginHandler)
	r.GET("/portal/user", func(c *gin.Context) {
		c.HTML(http.StatusOK, "user_portal.html", gin.H{"address": "0x00...", "balance": 0.0})
	})

	r.Run(":8085")
}

// --- Handlers ---
func loginHandler(c *gin.Context) {
	// If you aren't using variables, prefix them with underscore 
	// or use them to avoid build errors.
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/user"})
}
