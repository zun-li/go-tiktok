package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserIDKey = "user_id"

func Auth(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Authorization: Bearer <token> 中提取令牌。
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "请提供 Bearer 访问令牌",
			})
			return
		}

		// 2. 验证签名与有效期，只接受登录接口使用的 HS256 算法。
		var claims jwt.RegisteredClaims
		token, err := jwt.ParseWithClaims(parts[1], &claims, func(token *jwt.Token) (any, error) {
			return jwtSecret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "访问令牌无效或已过期，请重新登录",
			})
			return
		}

		// 3. sub 保存登录时签入的用户 ID。
		userID, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil || userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "访问令牌中的用户 ID 无效",
			})
			return
		}

		// 4. 将身份存入当前请求的上下文，继续执行业务接口。
		c.Set(UserIDKey, userID)
		c.Next()
	}
}
