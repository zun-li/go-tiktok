package http

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/account"
	"github.com/zun-li/go-tiktok/internal/middleware/jwt"
	"github.com/zun-li/go-tiktok/internal/middleware/rabbitmq"
	"github.com/zun-li/go-tiktok/internal/middleware/ratelimit"
	rediscache "github.com/zun-li/go-tiktok/internal/middleware/redis"
	"github.com/zun-li/go-tiktok/internal/social"
	"github.com/zun-li/go-tiktok/internal/video"
	"github.com/zun-li/go-tiktok/internal/worker"
	"gorm.io/gorm"
)

func registerInteractionRoutes(r *gin.Engine, db *gorm.DB, accounts *account.AccountService, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) {
	accountRepo := account.NewAccountRepository(db)
	videoRepo := video.NewVideoRepository(db)
	popularity, err := rabbitmq.NewPopularityMQ(rmq)
	if err != nil {
		log.Printf("Popularity queue disabled: %v", err)
	}
	likes, err := rabbitmq.NewLikeMQ(rmq)
	if err != nil {
		log.Printf("Like queue disabled: %v", err)
	}
	comments, err := rabbitmq.NewCommentMQ(rmq)
	if err != nil {
		log.Printf("Comment queue disabled: %v", err)
	}
	follows, err := rabbitmq.NewSocialMQ(rmq)
	if err != nil {
		log.Printf("Social queue disabled: %v", err)
	}
	likeHandler := video.NewLikeHandler(video.NewLikeService(video.NewLikeRepository(db), videoRepo, cache, likes, popularity))
	likeGroup := r.Group("/like", jwt.JWTAuth(accountRepo, cache))
	limiter := ratelimit.Limit(cache, "like_write", 30, time.Minute, ratelimit.KeyByAccount)
	likeGroup.POST("/like", limiter, likeHandler.Like)
	likeGroup.POST("/unlike", limiter, likeHandler.Unlike)
	likeGroup.POST("/isLiked", likeHandler.IsLiked)
	likeGroup.POST("/listMyLikedVideos", likeHandler.ListMyLikedVideos)
	commentHandler := video.NewCommentHandler(video.NewCommentService(video.NewCommentRepository(db), videoRepo, cache, comments, popularity), accounts)
	r.POST("/comment/listAll", commentHandler.GetAllComments)
	commentGroup := r.Group("/comment", jwt.JWTAuth(accountRepo, cache))
	commentGroup.POST("/publish", ratelimit.Limit(cache, "comment_write", 10, time.Minute, ratelimit.KeyByAccount), commentHandler.PublishComment)
	commentGroup.POST("/delete", commentHandler.DeleteComment)
	socialHandler := social.NewSocialHandler(social.NewSocialService(social.NewSocialRepository(db), accountRepo, follows, cache))
	socialGroup := r.Group("/social", jwt.JWTAuth(accountRepo, cache))
	socialLimiter := ratelimit.Limit(cache, "social_write", 20, time.Minute, ratelimit.KeyByAccount)
	socialGroup.POST("/follow", socialLimiter, socialHandler.Follow)
	socialGroup.POST("/unfollow", socialLimiter, socialHandler.Unfollow)
	socialGroup.POST("/getAllFollowers", socialHandler.GetAllFollowers)
	socialGroup.POST("/getAllVloggers", socialHandler.GetAllVloggers)
	socialGroup.POST("/getCounts", socialHandler.GetCounts)
	timeline, err := rabbitmq.NewTimelineMQ(rmq)
	if err != nil {
		log.Printf("Timeline queue disabled: %v", err)
	}
	worker.StartOutboxPoller(db, timeline)
	worker.StartConsumer(timeline, "video.timeline.update.queue", cache, rmq)
}
