package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Account structure with role and password fields
type Account struct {
	Address         string  `gorm:"primaryKey" json:"address"`
	Password        string  `json:"-"`
	Balance         float64 `gorm:"default:100.0" json:"balance"`
	Role            string  `gorm:"default:'user'" json:"role"` // 'user', 'miner', 'org', 'admin'
	PasswordChanged bool    `gorm:"default:false" json:"password_changed"`
}

var db *gorm.DB

func main() {
	gin.SetMode(gin.ReleaseMode)

	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	db.AutoMigrate(&Account{})

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// Health Check
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Public Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/explorer", func(c *gin.Context) { c.HTML(http.StatusOK, "explorer.html", nil) })
	r.GET("/docs", func(c *gin.Context) { c.HTML(http.StatusOK, "docs.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })
	r.GET("/core", func(c *gin.Context) { c.HTML(http.StatusOK, "core.html", nil) })
	r.GET("/markets", func(c *gin.Context) { c.HTML(http.StatusOK, "markets.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	// Portals
	r.GET("/portal/user", func(c *gin.Context) { c.HTML(http.StatusOK, "user_portal.html", nil) })
	r.GET("/portal/organization", func(c *gin.Context) { c.HTML(http.StatusOK, "organization_portal.html", nil) })
	r.GET("/portal/miner", func(c *gin.Context) { c.HTML(http.StatusOK, "miner_portal.html", nil) })

	// Auth & API
	r.POST("/auth/login", loginHandler)
	r.POST("/api/password/change", changePasswordHandler)
	r.POST("/api/transfer", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "success"}) })

	r.Run(":8085")
}

func loginHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")
	role := c.PostForm("role")

	var acc Account
	err := db.Where("address = ?", addr).First(&acc).Error
	if err != nil {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		acc = Account{Address: addr, Password: string(hashed), Role: role}
		db.Create(&acc)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + acc.Role})
}

func changePasswordHandler(c *gin.Context) {
	addr := c.PostForm("address")
	newPass := c.PostForm("new_password")

	var acc Account
	if err := db.Where("address = ?", addr).First(&acc).Error; err == nil {
		if acc.Role == "org" {
			hashed, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			db.Model(&acc).Updates(Account{Password: string(hashed), PasswordChanged: true})
			c.JSON(http.StatusOK, gin.H{"status": "Password updated"})
			return
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"status": "Unauthorized"})
}
