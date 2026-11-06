package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/handler"
	"github.com/zun-li/go-tiktok/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN 未设置")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接池失败: %v", err)
	}
	defer sqlDB.Close()

	log.Println("MySQL 连接成功")

	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("迁移用户表失败: %v", err)
	}

	router := gin.Default()

	router.POST("/account/register", handler.Register(db))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
