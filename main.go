package main

import (
	"fmt"
	"net/http"
	"os"

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

var db *gorm.DB

func main() {
	// Gin Setup
	gin.SetMode(gin.ReleaseMode)
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	db.AutoMigrate(&Account{})

	r := gin.Default()
	
	// Session Middleware Setup
	store := cookie.NewStore([]byte("secret-key-change-me"))
	r.Use(sessions.Sessions("mysession", store))

	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// 1. Navigation Routes
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
	r.GET("/explorer", func(c *gin.Context) { c.HTML(http.StatusOK, "explorer.html", nil) })
	r.GET("/docs", func(c *gin.Context) { c.HTML(http.StatusOK, "docs.html", nil) })
	r.GET("/ussd", func(c *gin.Context) { c.HTML(http.StatusOK, "ussd.html", nil) })
	r.GET("/core", func(c *gin.Context) { c.HTML(http.StatusOK, "core.html", nil) })
	r.GET("/markets", func(c *gin.Context) { c.HTML(http.StatusOK, "markets.html", nil) })
	r.GET("/news", func(c *gin.Context) { c.HTML(http.StatusOK, "news.html", nil) })

	// 2. Dynamic Portal Navigation (Use this link in your Nav bar)
	r.GET("/portal/my-portal", AuthRequired, func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role").(string)
		c.Redirect(http.StatusFound, "/portal/"+role)
	})

	// 3. Portal Routes (with Auth Check)
	portal := r.Group("/portal")
	portal.Use(AuthRequired)
	{
		portal.GET("/admin", func(c *gin.Context) { c.HTML(http.StatusOK, "admin_portal.html", gin.H{"role": "admin"}) })
		portal.GET("/user", func(c *gin.Context) { c.HTML(http.StatusOK, "user_portal.html", gin.H{"role": "user"}) })
		portal.GET("/organization", func(c *gin.Context) { c.HTML(http.StatusOK, "organization_portal.html", gin.H{"role": "organization"}) })
		portal.GET("/miner", func(c *gin.Context) { c.HTML(http.StatusOK, "miner_portal.html", gin.H{"role": "miner"}) })
	}

	// 4. Auth Handlers
	r.POST("/auth/login", loginHandler)
	r.POST("/auth/register", registerHandler)
	r.POST("/auth/recover", recoverHandler)
	r.GET("/auth/logout", logoutHandler)

	// 5. API Routes
	r.GET("/api/ledger", ledgerHandler)
	r.POST("/api/transfer", transferHandler)

	r.Run(":8085")
}

// Middleware: Verify Session
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("address") == nil {
		c.Redirect(http.StatusFound, "/news")
		c.Abort()
		return
	}
	c.Next()
}

// Handlers
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
	
	session := sessions.Default(c)
	session.Set("address", acc.Address)
	session.Set("role", acc.Role)
	session.Save()

	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/" + acc.Role})
}

func registerHandler(c *gin.Context) {
	addr := c.PostForm("address")
	pass := c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	db.Create(&Account{Address: addr, Password: string(hashed), Role: "user"})
	c.JSON(http.StatusOK, gin.H{"status": "success", "redirect": "/portal/user"})
}

func recoverHandler(c *gin.Context) {
	addr := c.PostForm("address")
	newPass := c.PostForm("password")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	db.Model(&Account{}).Where("address = ?", addr).Update("password", string(hashed))
	c.JSON(http.StatusOK, gin.H{"status": "Recovery successful", "redirect": "/news"})
}

func ledgerHandler(c *gin.Context) {
	var accounts []Account
	db.Find(&accounts)
	c.JSON(http.StatusOK, accounts)
}

func transferHandler(c *gin.Context) {
	receiver := c.PostForm("receiver")
	var amount float64
	fmt.Sscanf(c.PostForm("amount"), "%f", &amount)

	db.Model(&Account{}).Where("address = ?", "TREASURY_ROOT").Update("balance", gorm.Expr("balance - ?", amount))
	db.Model(&Account{}).Where("address = ?", receiver).Update("balance", gorm.Expr("balance + ?", amount))
	
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/news")
}
