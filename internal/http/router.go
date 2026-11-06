package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/account"
	"github.com/zun-li/go-tiktok/internal/middleware/jwt"
	"github.com/zun-li/go-tiktok/internal/middleware/rabbitmq"
	"github.com/zun-li/go-tiktok/internal/middleware/ratelimit"
	rediscache "github.com/zun-li/go-tiktok/internal/middleware/redis"
	"gorm.io/gorm"
	"time"
)

func SetRouter(db *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.Static("/static", "./.run/uploads")
	repo := account.NewAccountRepository(db)
	h := account.NewAccountHandler(account.NewAccountService(repo, cache))
	a := r.Group("/account")
	a.POST("/register", ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP), h.CreateAccount)
	a.POST("/login", ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP), h.Login)
	a.POST("/refresh", h.Refresh)
	a.POST("/changePassword", h.ChangePassword)
	a.POST("/findByID", h.FindByID)
	a.POST("/findByUsername", h.FindByUsername)
	protected := a.Group("", jwt.JWTAuth(repo, cache))
	protected.GET("/me", h.Me)
	protected.POST("/logout", h.Logout)
	protected.POST("/rename", h.Rename)
	protected.POST("/uploadAvatar", h.UploadAvatar)
	protected.POST("/updateProfile", h.UpdateProfile)
	return r
}
