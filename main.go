package main

import (
	"html/template" // Required for FuncMap
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var jwtKey = []byte("SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET")

type Account struct {
	Address         string  `gorm:"primaryKey" json:"address"`
	Balance         float64 `gorm:"default:0.0" json:"balance"`
	PasswordChanged bool    `gorm:"default:false" json:"password_changed"`
	IsOrg           bool    `gorm:"default:false" json:"is_org"`
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

	// Register the 'loop' function for templates
	r.SetFuncMap(template.FuncMap{
		"loop": func(n int) []int {
			res := make([]int, n)
			for i := 0; i < n; i++ {
				res[i] = i
			}
			return res
		},
	})

	// 1. Serve static files and templates
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// 2. Frontend Routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"is_logged_in": false,
			"total_supply": "1,250,000",
			"total_nodes":  "48",
		})
	})

	// 3. API Routes
	r.POST("/auth/login", loginHandler)
	r.POST("/api/transfer", authMiddleware(), transferHandler)
	r.GET("/api/ai-monitor", aiMonitorHandler)
	r.GET("/api/balances", func(c *gin.Context) {
		var accounts []Account
		db.Limit(5).Find(&accounts)
		accMap := make(map[string]float64)
		for _, acc := range accounts {
			accMap[acc.Address] = acc.Balance
		}
		c.JSON(http.StatusOK, gin.H{"accounts": accMap})
	})

	r.Run(":8085")
}

// --- Middleware & Handlers remain the same ---

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return jwtKey, nil })
		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Set("address", claims["address"])
				c.Set("role", claims["role"])
				c.Next()
				return
			}
		}
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func loginHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")
	var acc Account
	db.FirstOrCreate(&acc, Account{Address: addr})
	role := "User"
	if pass == "admin123" { role = "Admin" }
	if pass == "Organization@portal" { role = "Organization"; acc.IsOrg = true; db.Save(&acc) }
	if pass == "miner123" { role = "Miner" }
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"address": addr,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 168).Unix(),
	})
	tokenStr, _ := token.SignedString(jwtKey)
	c.JSON(http.StatusOK, gin.H{"token": tokenStr, "redirect": "/portal/" + role})
}

func transferHandler(c *gin.Context) {
	senderAddr, _ := c.Get("address")
	role, _ := c.Get("role")
	var input struct {
		Recipient string `json:"recipient"`
		Amount    float64 `json:"amount"`
	}
	c.ShouldBindJSON(&input)
	err := db.Transaction(func(tx *gorm.DB) error {
		if role != "Admin" {
			var sender Account
			tx.First(&sender, "address = ?", senderAddr)
			if sender.Balance < input.Amount { return http.ErrAbortHandler }
			tx.Model(&sender).Update("balance", sender.Balance-input.Amount)
		}
		var recipient Account
		tx.FirstOrCreate(&recipient, Account{Address: input.Recipient})
		tx.Model(&recipient).Update("balance", recipient.Balance+input.Amount)
		return nil
	})
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status": "error"}) }
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func aiMonitorHandler(c *gin.Context) {
	var malicious []Account
	db.Where("balance < ?", 0).Find(&malicious)
	db.Model(&Account{}).Where("balance < ?", 0).Update("balance", 0)
	c.JSON(http.StatusOK, gin.H{"malicious_detected": len(malicious) > 0})
}
