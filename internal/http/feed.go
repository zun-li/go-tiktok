package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/account"
	"github.com/zun-li/go-tiktok/internal/feed"
	"github.com/zun-li/go-tiktok/internal/middleware/jwt"
	rediscache "github.com/zun-li/go-tiktok/internal/middleware/redis"
	"github.com/zun-li/go-tiktok/internal/social"
	"github.com/zun-li/go-tiktok/internal/video"
	"gorm.io/gorm"
)

func registerFeedRoutes(r *gin.Engine, db *gorm.DB, accounts *account.AccountService, cache *rediscache.Client) {
	accountRepo := account.NewAccountRepository(db)
	videos := video.NewVideoRepository(db)
	follows := social.NewSocialRepository(db)
	h := feed.NewFeedHandler(feed.NewFeedService(feed.NewFeedRepository(db), video.NewLikeRepository(db), cache))
	group := r.Group("/feed", jwt.SoftJWTAuth(accountRepo, cache))
	group.POST("/listLatest", h.ListLatest)
	group.POST("/listLikesCount", h.ListLikesCount)
	group.POST("/listByPopularity", h.ListByPopularity)
	group.POST("/listByTag", h.ListByTag)
	group.POST("/listByFollowing", jwt.JWTAuth(accountRepo, cache), h.ListByFollowing)
	r.POST("/account/getProfile", func(c *gin.Context) {
		var req account.GetProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.AccountID == 0 {
			c.JSON(400, gin.H{"error": "account_id is required"})
			return
		}
		acc, err := accounts.FindByID(c.Request.Context(), req.AccountID)
		if err != nil {
			c.JSON(404, gin.H{"error": "account not found"})
			return
		}
		ctx := c.Request.Context()
		videoCount, err := videos.CountByAuthor(ctx, req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to load profile"})
			return
		}
		totalLikes, err := videos.TotalLikesByAuthor(ctx, req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to load profile"})
			return
		}
		followers, err := follows.CountFollowers(ctx, req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to load profile"})
			return
		}
		following, err := follows.CountVloggers(ctx, req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to load profile"})
			return
		}
		c.JSON(200, account.GetProfileResponse{Account: account.FindByIDResponse{ID: acc.ID, Username: acc.Username, AvatarURL: acc.AvatarURL, Bio: acc.Bio}, VideoCount: videoCount, TotalLikes: totalLikes, FollowerCount: followers, VloggerCount: following})
	})
}
