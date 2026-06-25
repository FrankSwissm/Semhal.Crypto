package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Account struct {
	Address         string  `gorm:"primaryKey" json:"address"`
	Password        string  `json:"-"`
	Balance         float64 `gorm:"default:100.0" json:"balance"`
	Role            string  `gorm:"default:'user'" json:"role"`
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

	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	// Auth Handlers
	r.POST("/auth/login", loginHandler)
	r.POST("/auth/register", registerHandler)
	r.POST("/auth/recover", recoverHandler)

	r.Run(":8085")
}

func loginHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")
	var acc Account
	if err := db.Where("address = ?", addr).First(&acc).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(pass)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + acc.Role})
}

func registerHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	db.Create(&Account{Address: addr, Password: string(hashed), Role: "user"})
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func recoverHandler(c *gin.Context) {
	addr := c.PostForm("address")
	newPass := c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	db.Model(&Account{}).Where("address = ?", addr).Update("password", string(hashed))
	c.JSON(http.StatusOK, gin.H{"status": "Recovery successful"})
}
