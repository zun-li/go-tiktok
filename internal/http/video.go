package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/account"
	"github.com/zun-li/go-tiktok/internal/middleware/jwt"
	"github.com/zun-li/go-tiktok/internal/middleware/rabbitmq"
	rediscache "github.com/zun-li/go-tiktok/internal/middleware/redis"
	"github.com/zun-li/go-tiktok/internal/video"
	"gorm.io/gorm"
	"log"
)

func registerVideoRoutes(r *gin.Engine, db *gorm.DB, accounts *account.AccountService, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) {
	repo := video.NewVideoRepository(db)
	popularity, err := rabbitmq.NewPopularityMQ(rmq)
	if err != nil {
		log.Printf("Popularity queue disabled: %v", err)
	}
	h := video.NewVideoHandler(video.NewVideoService(repo, cache, popularity), accounts)
	chunks := video.NewChunkUploadHandler(cache)
	group := r.Group("/video")
	group.POST("/listByAuthorID", h.ListByAuthorID)
	group.POST("/getDetail", h.GetDetail)
	protected := group.Group("", jwt.JWTAuth(account.NewAccountRepository(db), cache))
	protected.POST("/uploadVideo", h.UploadVideo)
	protected.POST("/uploadCover", h.UploadCover)
	protected.POST("/publish", h.PublishVideo)
	protected.POST("/chunk/init", chunks.InitChunkUpload)
	protected.POST("/chunk/upload", chunks.UploadChunk)
	protected.POST("/chunk/status", chunks.ChunkStatus)
	protected.POST("/chunk/complete", chunks.CompleteChunkUpload)
}
