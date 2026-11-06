package rabbitmq

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRabbitMQRouting(t *testing.T) {
	endpoint := os.Getenv("TEST_RABBITMQ_URL")
	if endpoint == "" {
		t.Skip("set TEST_RABBITMQ_URL or run the Compose tests service")
	}
	uri, err := amqp.ParseURI(endpoint)
	if err != nil || !strings.HasSuffix(uri.Vhost, "_test") {
		t.Fatal("TEST_RABBITMQ_URL must use a virtual host ending in _test")
	}
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	id, err := newEventID(8)
	if err != nil {
		t.Fatal(err)
	}
	exchange, queue := "test."+id, "test."+id
	otherQueue := queue + ".other"
	if err := DeclareTopic(ch, exchange, queue, "event.created"); err != nil {
		t.Fatal(err)
	}
	if err := DeclareTopic(ch, exchange, otherQueue, "event.other"); err != nil {
		t.Fatal(err)
	}
	defer ch.ExchangeDelete(exchange, false, false)
	defer ch.QueueDelete(queue, false, false, false)
	defer ch.QueueDelete(queue+".dlx", false, false, false)
	defer ch.QueueDelete(otherQueue, false, false, false)
	defer ch.QueueDelete(otherQueue+".dlx", false, false, false)
	get := func(name string) amqp.Delivery {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			delivery, ok, err := ch.Get(name, false)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				return delivery
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatalf("no delivery in queue %s", name)
		return amqp.Delivery{}
	}
	if err := PublishJSON(context.Background(), ch, exchange, "event.created", map[string]string{"event_id": id}); err != nil {
		t.Fatal(err)
	}
	delivery := get(queue)
	var data map[string]string
	if err := json.Unmarshal(delivery.Body, &data); err != nil || data["event_id"] != id {
		t.Fatalf("invalid JSON delivery: %s", delivery.Body)
	}
	if delivery.DeliveryMode != amqp.Persistent {
		t.Fatal("event is not persistent")
	}
	if err := delivery.Nack(false, false); err != nil {
		t.Fatal(err)
	}
	dead := get(queue + ".dlx")
	if err := dead.Ack(false); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := ch.Get(otherQueue+".dlx", true); err != nil || ok {
		t.Fatalf("dead letter routed to unrelated queue: ok=%v err=%v", ok, err)
	}
}
