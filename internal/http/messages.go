package http

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zun-li/go-tiktok/internal/account"
	"github.com/zun-li/go-tiktok/internal/message"
	"github.com/zun-li/go-tiktok/internal/middleware/jwt"
	"github.com/zun-li/go-tiktok/internal/middleware/rabbitmq"
	rediscache "github.com/zun-li/go-tiktok/internal/middleware/redis"
	"github.com/zun-li/go-tiktok/internal/worker"
	"gorm.io/gorm"
)

func registerMessageRoutes(r *gin.Engine, db *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) {
	h := message.NewHandler(message.NewService(message.NewRepository(db)))
	group := r.Group("/message", jwt.JWTAuth(account.NewAccountRepository(db), cache))
	group.POST("/send", h.Send)
	group.POST("/list", h.List)
	hub := worker.NewSSEHub(db)
	notifications := r.Group("/notification", hub.SSERequireAuth())
	hub.RegisterRoutes(r, notifications)
	if rmq == nil {
		return
	}
	ch, err := rmq.NewChannel()
	if err != nil {
		log.Printf("Notification queues disabled: %v", err)
		return
	}
	defer ch.Close()
	for _, topic := range []struct{ exchange, queue, key string }{
		{"like.events", "notification.like", "like.like"},
		{"comment.events", "notification.comment", "comment.publish"},
		{"social.events", "notification.social", "social.follow"},
	} {
		if err := rabbitmq.DeclareTopic(ch, topic.exchange, topic.queue, topic.key); err != nil {
			log.Printf("Notification queue disabled: %v", err)
			continue
		}
		go func(queue string) {
			for {
				consumer, err := rmq.NewChannel()
				if err == nil {
					_ = consumer.Qos(50, 0, false)
					err = worker.NewNotificationWorker(consumer, db, queue, hub).Run(context.Background())
					_ = consumer.Close()
				}
				log.Printf("Notification consumer %s restarting: %v", queue, err)
				time.Sleep(5 * time.Second)
			}
		}(topic.queue)
	}
}
