package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/zun-li/go-tiktok/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const accessTokenTTL = 2 * time.Hour

type loginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

func Login(db *gorm.DB, jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 校验请求参数。
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求格式或参数不符合要求",
			})
			return
		}

		if len(req.Password) > 72 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "密码不能超过 72 字节",
			})
			return
		}

		// 2. 按用户名查询用户，沿用注册时的小写规则。
		var user model.User
		err := db.WithContext(c.Request.Context()).
			Where("username = ?", strings.ToLower(req.Username)).
			First(&user).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "用户名或密码错误",
			})
			return
		}
		if err != nil {
			log.Printf("查询登录用户失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "登录失败，请稍后重试",
			})
			return
		}

		// 3. 使用数据库中的哈希校验密码。
		err = bcrypt.CompareHashAndPassword(
			[]byte(user.PasswordHash),
			[]byte(req.Password),
		)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "用户名或密码错误",
			})
			return
		}

		// 4. 密码正确，签发有效期为两小时的访问令牌。
		now := time.Now()
		claims := jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		accessToken, err := token.SignedString(jwtSecret)
		if err != nil {
			log.Printf("签发访问令牌失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "登录失败，请稍后重试",
			})
			return
		}

		// 5. 返回令牌和使用方式。
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   int(accessTokenTTL.Seconds()),
		})
	}
}
