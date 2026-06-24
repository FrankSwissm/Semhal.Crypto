package main

import (
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Models ---
type Account struct {
	Address string  `gorm:"primaryKey" json:"address"`
	Balance float64 `gorm:"default:0.0" json:"balance"`
	IsOrg   bool    `gorm:"default:false" json:"is_org"`
}

type Transaction struct {
	Recipient string
	Amount    float64
	Status    string
}

type MarketItem struct {
	Name       string
	Price      string
	Delta      string
	IsNegative bool
}

var db *gorm.DB
var jwtKey = []byte("SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET")

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

	// --- Public Routes ---
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"total_supply": "1,250,000", "total_nodes": "48"})
	})

	r.GET("/explorer", func(c *gin.Context) {
		var accounts []Account
		db.Find(&accounts)
		c.HTML(http.StatusOK, "explorer.html", gin.H{"ledger": accounts})
	})

	r.GET("/markets", func(c *gin.Context) {
		c.HTML(http.StatusOK, "markets.html", gin.H{
			"crypto_data": []MarketItem{
				{Name: "BTC / Bitcoin", Price: "$96,430.50", Delta: "+1.82%"},
			},
		})
	})

	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })

	// --- Auth & Portal Routes ---
	r.POST("/auth/login", loginHandler)
	
	r.GET("/portal/user", authMiddleware(), func(c *gin.Context) {
		addr, _ := c.Get("address")
		var acc Account
		db.First(&acc, "address = ?", addr)
		c.HTML(http.StatusOK, "user_portal.html", gin.H{"address": acc.Address, "balance": acc.Balance})
	})

	r.GET("/portal/organization", authMiddleware(), func(c *gin.Context) {
		addr, _ := c.Get("address")
		var acc Account
		db.First(&acc, "address = ?", addr)
		c.HTML(http.StatusOK, "organization_portal.html", gin.H{"address": acc.Address, "balance": acc.Balance})
	})

	r.GET("/portal/miner", authMiddleware(), func(c *gin.Context) {
		addr, _ := c.Get("address")
		c.HTML(http.StatusOK, "miner_portal.html", gin.H{"address": addr, "balance": 1000.0})
	})

	// --- APIs ---
	r.POST("/api/transfer", authMiddleware(), transferHandler)
	r.GET("/api/ai-monitor", aiMonitorHandler)

	r.Run(":8085")
}

// --- Middleware & Handlers ---

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization") // Or read from Cookie
		// Validation logic...
		c.Next()
	}
}

func loginHandler(c *gin.Context) {
	addr := c.PostForm("address")
	role := "user"
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + role})
}

func transferHandler(c *gin.Context) {
	sender, _ := c.Get("address")
	amount, _ := strconv.ParseFloat(c.PostForm("amount"), 64)
	recipient := c.PostForm("recipient")
	// Database Logic here...
	c.JSON(http.StatusOK, gin.H{"status": "success", "new_balance": 500.0})
}

func aiMonitorHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"malicious_detected": false})
}
