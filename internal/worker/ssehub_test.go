package worker

import (
	"bufio"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSSEDeliversOnlyRecipientEvents(t *testing.T) {
	hub := NewSSEHub(nil)
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) { c.Set("accountID", uint(42)); hub.SSEHandler(c) })
	server := httptest.NewServer(router)
	defer server.Close()
	client := server.Client()
	client.Timeout = 3 * time.Second
	response, err := client.Get(server.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal("invalid SSE response")
	}
	hub.Push(99, &Notification{ID: 1, RecipientID: 99})
	hub.Push(42, &Notification{ID: 2, RecipientID: 42, Type: "like"})
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "data: ") {
		t.Fatalf("unexpected event: %s", line)
	}
	var event Notification
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
		t.Fatal(err)
	}
	if event.ID != 2 || event.RecipientID != 42 {
		t.Fatalf("wrong recipient event: %+v", event)
	}
}
