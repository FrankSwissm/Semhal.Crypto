package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Models
type Account struct {
	Address  string  `gorm:"primaryKey" json:"address"`
	Password string  `json:"-"`
	Balance  float64 `gorm:"default:100.0" json:"balance"`
	Role     string  `gorm:"default:'user'" json:"role"` // 'user', 'miner', 'org', 'admin'
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

	// Navigation Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	// ... (Add your other nav routes here: /explorer, /markets, /news, etc.)

	// Auth & Portal Routes
	r.POST("/auth/login", loginHandler)
	r.GET("/portal/:role", portalHandler)

	// API Routes
	r.POST("/api/transfer", transferHandler)
	r.GET("/api/ai-monitor", aiMonitorHandler)

	r.Run(":8085")
}

// Handlers
func loginHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")

	var acc Account
	if err := db.Where("address = ?", addr).First(&acc).Error; err != nil {
		// Auto-register logic for demonstration
		hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		acc = Account{Address: addr, Password: string(hashed), Role: "user"}
		db.Create(&acc)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + acc.Role})
}

func portalHandler(c *gin.Context) {
	role := c.Param("role")
	templateName := role + "_portal.html"
	
	// Fetch account data
	var acc Account
	db.First(&acc, "role = ?", role)

	c.HTML(http.StatusOK, templateName, gin.H{
		"address": acc.Address,
		"balance": acc.Balance,
	})
}

func transferHandler(c *gin.Context) {
	var input struct {
		Sender    string  `json:"sender"`
		Recipient string  `json:"recipient"`
		Amount    float64 `json:"amount"`
	}
	c.ShouldBindJSON(&input)

	err := db.Transaction(func(tx *gorm.DB) error {
		var sender, recipient Account
		tx.First(&sender, "address = ?", input.Sender)
		tx.FirstOrCreate(&recipient, Account{Address: input.Recipient})

		if sender.Balance < input.Amount { return gorm.ErrInvalidData }
		
		tx.Model(&sender).Update("balance", sender.Balance-input.Amount)
		tx.Model(&recipient).Update("balance", recipient.Balance+input.Amount)
		return nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Transaction failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func aiMonitorHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"malicious_detected": false})
}
