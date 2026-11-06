package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/middleware"
	"github.com/zun-li/go-tiktok/internal/model"
	"gorm.io/gorm"
)

func Me(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint64(middleware.UserIDKey)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}

		// 只查询公开字段，密码哈希不参与当前用户信息的返回。
		var user model.User
		err := db.WithContext(c.Request.Context()).
			Select("id", "username", "created_at").
			First(&user, userID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在，请重新登录"})
			return
		}
		if err != nil {
			log.Printf("查询当前用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败，请稍后重试"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
