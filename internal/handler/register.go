package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 解析 JSON，并执行 binding 标签中的校验。
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求格式或参数不符合要求",
			})
			return
		}

		// 2. bcrypt 限制输入最多为 72 字节。
		if len(req.Password) > 72 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "密码不能超过 72 字节",
			})
			return
		}

		// 3. 将明文密码转换成哈希。
		hash, err := bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			log.Printf("生成密码哈希失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "注册失败，请稍后重试",
			})
			return
		}

		// 4. 构造用户，只将密码哈希写入数据库。
		user := model.User{
			Username:     strings.ToLower(req.Username),
			PasswordHash: string(hash),
		}

		err = db.WithContext(c.Request.Context()).Create(&user).Error
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "用户名已存在",
			})
			return
		}
		if err != nil {
			log.Printf("创建用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "注册失败，请稍后重试",
			})
			return
		}

		// 5. 插入成功后，GORM 已将生成的 ID 填回 user。
		c.JSON(http.StatusCreated, gin.H{
			"id":       user.ID,
			"username": user.Username,
		})
	}
}
