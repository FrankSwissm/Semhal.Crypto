package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Account Model
type Account struct {
	Address         string  `gorm:"primaryKey" json:"address"`
	Password        string  `json:"-"`
	Balance         float64 `gorm:"default:100.0" json:"balance"`
	Role            string  `gorm:"default:'user'" json:"role"`
	PasswordChanged bool    `gorm:"default:false" json:"password_changed"`
}

// Transaction History Model
type Transaction struct {
	ID        uint      `gorm:"primaryKey"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Amount    float64   `json:"amount"`
	Exchange  string    `json:"exchange"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	db        *gorm.DB
	RateCache = map[string]float64{"semhal": 1.0, "binance": 1.02, "coinbase": 1.03, "kraken": 1.02}
	mu        sync.RWMutex
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	db.AutoMigrate(&Account{}, &Transaction{})

	if err := db.Where("address = ?", "TREASURY_ROOT").First(&Account{}).Error; err != nil {
		db.Create(&Account{Address: "TREASURY_ROOT", Balance: 48217477500.0, Role: "admin"})
	}

	go StartOracleWorker()

	r := gin.Default()
	store := cookie.NewStore([]byte("secret-key-change-me"))
	r.Use(sessions.Sessions("mysession", store))

	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/portfolio", AuthRequired, func(c *gin.Context) { c.HTML(http.StatusOK, "portfolio.html", nil) })
	r.GET("/explorer", func(c *gin.Context) { c.HTML(http.StatusOK, "explorer.html", nil) })
	r.GET("/transactions", AuthRequired, func(c *gin.Context) { c.HTML(http.StatusOK, "history.html", nil) })
	r.GET("/docs", func(c *gin.Context) { c.HTML(http.StatusOK, "docs.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })
	r.GET("/core", func(c *gin.Context) { c.HTML(http.StatusOK, "core.html", nil) })
	r.GET("/markets", func(c *gin.Context) { c.HTML(http.StatusOK, "markets.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	r.GET("/portal/my-portal", AuthRequired, func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role").(string)
		c.Redirect(http.StatusFound, "/portal/"+role)
	})

	portal := r.Group("/portal")
	portal.Use(AuthRequired)
	{
		portal.GET("/admin", func(c *gin.Context) { c.HTML(http.StatusOK, "admin_portal.html", gin.H{"role": "admin"}) })
		portal.GET("/user", func(c *gin.Context) {
			session := sessions.Default(c)
			addr := session.Get("address").(string)
			var acc Account
			db.Where("address = ?", addr).First(&acc)
			c.HTML(http.StatusOK, "user_portal.html", gin.H{"role": "user", "address": acc.Address, "balance": acc.Balance})
		})
	}

	r.POST("/auth/login", loginHandler)
	r.POST("/auth/register", registerHandler)
	r.POST("/auth/recover", recoverHandler)
	r.GET("/auth/logout", logoutHandler)
	r.GET("/api/ledger", ledgerHandler)
	r.POST("/api/transfer", transferHandler)
	r.GET("/api/history", AuthRequired, historyHandler)

	r.Run(":8085")
}

// Oracle Infrastructure with Live Fetching
func StartOracleWorker() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		// Mock implementation of API integration - replace with real endpoint
		// e.g., "https://api.exchange.com/v1/ticker"
		mu.Lock()
		RateCache["binance"] = FetchLiveRate("binance")
		RateCache["coinbase"] = FetchLiveRate("coinbase")
		mu.Unlock()
	}
}

func FetchLiveRate(exchange string) float64 {
	// In a real implementation, perform HTTP GET request here
	// This maintains the "consensus" logic of our sovereign node
	return 1.02 // Placeholder for real-time data
}

func transferHandler(c *gin.Context) {
	session := sessions.Default(c)
	senderAddr := session.Get("address").(string)
	receiver, exchange := c.PostForm("recipient"), c.PostForm("exchange")
	var amount float64
	fmt.Sscanf(c.PostForm("amount"), "%f", &amount)

	// API-Driven Settlement: Validate against current rate
	currentRate := GetRate(exchange)
	effectiveAmount := amount * currentRate

	// Verification: Atomic settlement attempt
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Check liquidity and deduct
		if err := tx.Model(&Account{}).Where("address = ? AND balance >= ?", senderAddr, effectiveAmount).
			Update("balance", gorm.Expr("balance - ?", effectiveAmount)).Error; err != nil {
			return err
		}
		// 2. Settlement log
		tx.Create(&Transaction{Sender: senderAddr, Receiver: receiver, Amount: effectiveAmount, Exchange: exchange, CreatedAt: time.Now()})
		// 3. Complete transfer
		return tx.Model(&Account{}).Where("address = ?", receiver).Update("balance", gorm.Expr("balance + ?", effectiveAmount)).Error
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "Settlement failed: Liquidity or Rate mismatch"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "settled_amount": effectiveAmount})
}

// Keep existing Handlers: loginHandler, registerHandler, recoverHandler, ledgerHandler, historyHandler, AuthRequired, logoutHandler
func AuthRequired(c *gin.Context) { session := sessions.Default(c); if session.Get("address") == nil { c.Redirect(http.StatusFound, "/news"); c.Abort(); return }; c.Next() }
func loginHandler(c *gin.Context) { /*...*/ }
func registerHandler(c *gin.Context) { /*...*/ }
func recoverHandler(c *gin.Context) { /*...*/ }
func ledgerHandler(c *gin.Context) { /*...*/ }
func historyHandler(c *gin.Context) { /*...*/ }
func logoutHandler(c *gin.Context) { /*...*/ }
func GetRate(exchange string) float64 { mu.RLock(); defer mu.RUnlock(); return RateCache[exchange] }
