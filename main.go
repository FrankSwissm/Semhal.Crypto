package main

import (
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

// Models
type Account struct {
	Address         string  `gorm:"primaryKey" json:"address"`
	Password        string  `json:"-"`
	Balance         float64 `gorm:"default:100.0" json:"balance"`
	Role            string  `gorm:"default:'user'" json:"role"`
	PasswordChanged bool    `gorm:"default:false" json:"password_changed"`
}

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
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
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
	r.POST("/api/transfer", transferHandler)
	r.GET("/api/history", historyHandler)
	r.Run(":8085")
}

// Oracle Worker with Aggregation
func StartOracleWorker() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		// Aggregation logic: Take multiple simulated sources and find the median
		sources := []float64{1.01, 1.02, 1.03} 
		sort.Float64s(sources)
		median := sources[len(sources)/2]

		mu.Lock()
		for k := range RateCache {
			if k != "semhal" { RateCache[k] = median }
		}
		mu.Unlock()
	}
}

func GetRate(exchange string) float64 {
	mu.RLock()
	defer mu.RUnlock()
	if rate, ok := RateCache[exchange]; ok { return rate }
	return 1.0
}

func transferHandler(c *gin.Context) {
	session := sessions.Default(c)
	senderAddr := session.Get("address").(string)
	receiver := c.PostForm("recipient")
	exchange := c.PostForm("exchange")
	var amount float64
	fmt.Sscanf(c.PostForm("amount"), "%f", &amount)

	effectiveAmount := amount * GetRate(exchange)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Account{}).Where("address = ? AND balance >= ?", senderAddr, effectiveAmount).
			Update("balance", gorm.Expr("balance - ?", effectiveAmount)).Error; err != nil {
			return err
		}
		tx.Create(&Transaction{Sender: senderAddr, Receiver: receiver, Amount: effectiveAmount, Exchange: exchange, CreatedAt: time.Now()})
		return tx.Model(&Account{}).Where("address = ?", receiver).Update("balance", gorm.Expr("balance + ?", effectiveAmount)).Error
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "Transaction failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func historyHandler(c *gin.Context) {
	var txs []Transaction
	db.Order("created_at desc").Limit(10).Find(&txs)
	c.JSON(http.StatusOK, txs)
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

// ... Keep existing login, register, recover, ledger, logout handlers as before
