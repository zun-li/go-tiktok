package http_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/zun-li/go-tiktok/internal/account"
	appdb "github.com/zun-li/go-tiktok/internal/db"
	apphttp "github.com/zun-li/go-tiktok/internal/http"
	"github.com/zun-li/go-tiktok/internal/message"
	"github.com/zun-li/go-tiktok/internal/video"
	"github.com/zun-li/go-tiktok/internal/worker"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response %s: %v", body, err)
	}
	return value
}

// TEST_MYSQL_DSN must point to a database ending in _test. Each run creates
// and removes a separate database, leaving application data untouched.
func TestIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set TEST_MYSQL_DSN or run the Compose tests service")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+_test$`).MatchString(cfg.DBName) {
		t.Fatal("TEST_MYSQL_DSN database must end in _test")
	}
	database := fmt.Sprintf("%s_%d", cfg.DBName, time.Now().UnixNano())
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE `"+database+"` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + database + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName = database
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer appdb.CloseDB(db)
	if err := appdb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", "integration-signing-secret-at-least-32-bytes")
	t.Chdir(t.TempDir())
	router := apphttp.SetRouter(db, nil, nil)
	request := func(method, path string, body any, token string, expected int) []byte {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != expected {
			t.Fatalf("%s %s: status=%d expected=%d body=%s", method, path, res.Code, expected, res.Body.String())
		}
		return res.Body.Bytes()
	}
	upload := func(path, filename string, content []byte, token string) string {
		t.Helper()
		var buf bytes.Buffer
		form := multipart.NewWriter(&buf)
		file, err := form.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", path, &buf)
		req.Header.Set("Content-Type", form.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("upload %s: %d %s", path, res.Code, res.Body.String())
		}
		data := decodeJSON[map[string]string](t, res.Body.Bytes())
		for _, key := range []string{"play_url", "cover_url", "avatar_url"} {
			if data[key] != "" {
				return data[key]
			}
		}
		t.Fatal("upload returned no URL")
		return ""
	}
	credentials := func(name, password string) map[string]string {
		return map[string]string{"username": name, "password": password}
	}
	request("POST", "/account/register", credentials("alice", "password123"), "", 200)
	request("POST", "/account/register", credentials("bob", "password123"), "", 200)
	request("POST", "/account/register", credentials("alice", "password123"), "", 409)
	request("POST", "/account/register", credentials("ab", "short"), "", 400)
	request("POST", "/account/login", credentials("alice", "wrongpassword"), "", 401)
	alice := decodeJSON[account.LoginResponse](t, request("POST", "/account/login", credentials("ALICE", "password123"), "", 200))
	bob := decodeJSON[account.LoginResponse](t, request("POST", "/account/login", credentials("bob", "password123"), "", 200))
	me := request("GET", "/account/me", nil, alice.Token, 200)
	if strings.Contains(string(me), "password") || strings.Contains(string(me), "refresh_token") {
		t.Fatal("private account fields exposed")
	}
	refreshed := decodeJSON[account.LoginResponse](t, request("POST", "/account/refresh", map[string]string{"refresh_token": alice.RefreshToken}, "", 200))
	request("GET", "/account/me", nil, alice.Token, 401)
	alice.Token = refreshed.Token
	renamed := decodeJSON[account.LoginResponse](t, request("POST", "/account/rename", map[string]string{"new_username": "alice2"}, alice.Token, 200))
	request("GET", "/account/me", nil, alice.Token, 401)
	alice.Token = renamed.Token
	avatar := upload("/account/uploadAvatar", "avatar.png", []byte("avatar fixture"), alice.Token)
	request("POST", "/account/updateProfile", map[string]string{"bio": "test profile", "avatar_url": avatar}, alice.Token, 200)
	media := []byte("video upload fixture")
	play := upload("/video/uploadVideo", "video.mp4", media, alice.Token)
	cover := upload("/video/uploadCover", "cover.png", []byte("cover fixture"), alice.Token)
	parsed, err := url.Parse(play)
	if err != nil {
		t.Fatal(err)
	}
	downloaded := request("GET", parsed.Path, nil, "", 200)
	if !bytes.Equal(downloaded, media) {
		t.Fatal("uploaded video content changed")
	}
	published := decodeJSON[video.Video](t, request("POST", "/video/publish", map[string]string{"title": "hello #golang", "description": "test video", "play_url": play, "cover_url": cover}, alice.Token, 200))
	input := map[string]uint{"video_id": published.ID}
	request("POST", "/video/getDetail", map[string]uint{"id": published.ID}, "", 200)
	request("POST", "/video/listByAuthorID", map[string]uint{"author_id": alice.AccountID}, "", 200)
	for _, path := range []string{"/feed/listLatest", "/feed/listLikesCount", "/feed/listByPopularity"} {
		data := request("POST", path, map[string]int{"limit": 10}, "", 200)
		var response struct {
			Videos []json.RawMessage `json:"video_list"`
		}
		response = decodeJSON[struct {
			Videos []json.RawMessage `json:"video_list"`
		}](t, data)
		if len(response.Videos) != 1 {
			t.Fatalf("%s did not return the published video: %s", path, data)
		}
	}
	request("POST", "/feed/listByTag", map[string]any{"tag_name": "golang", "limit": 10}, "", 200)
	request("POST", "/social/follow", map[string]uint{"vlogger_id": alice.AccountID}, bob.Token, 200)
	following := decodeJSON[map[string]any](t, request("POST", "/feed/listByFollowing", map[string]int{"limit": 10}, bob.Token, 200))
	if len(following["video_list"].([]any)) != 1 {
		t.Fatal("following feed is empty")
	}
	request("POST", "/social/getAllFollowers", map[string]uint{"vlogger_id": alice.AccountID}, bob.Token, 200)
	request("POST", "/social/getAllVloggers", map[string]uint{"follower_id": bob.AccountID}, bob.Token, 200)
	request("POST", "/social/getCounts", map[string]int{}, bob.Token, 200)
	request("POST", "/like/like", input, bob.Token, 200)
	liked := decodeJSON[map[string]bool](t, request("POST", "/like/isLiked", input, bob.Token, 200))
	if !liked["is_liked"] {
		t.Fatal("like was not saved")
	}
	request("POST", "/like/listMyLikedVideos", map[string]int{}, bob.Token, 200)
	request("POST", "/like/unlike", input, bob.Token, 200)
	request("POST", "/comment/publish", map[string]any{"video_id": published.ID, "content": "hello @bob"}, alice.Token, 200)
	comments := decodeJSON[[]video.Comment](t, request("POST", "/comment/listAll", input, "", 200))
	if len(comments) != 1 {
		t.Fatal("comment was not saved")
	}
	request("POST", "/comment/delete", map[string]uint{"comment_id": comments[0].ID}, bob.Token, 401)
	request("POST", "/comment/delete", map[string]uint{"comment_id": comments[0].ID}, alice.Token, 200)
	request("POST", "/account/getProfile", map[string]uint{"account_id": alice.AccountID}, "", 200)
	sent := decodeJSON[message.Message](t, request("POST", "/message/send", map[string]any{"to_id": bob.AccountID, "content": "hello"}, alice.Token, 200))
	messages := decodeJSON[message.ListResponse](t, request("POST", "/message/list", map[string]uint{"peer_id": alice.AccountID}, bob.Token, 200))
	if len(messages.Messages) != 1 || messages.Messages[0].ID != sent.ID {
		t.Fatal("message was not saved")
	}
	notification := worker.Notification{RecipientID: bob.AccountID, SenderID: alice.AccountID, Type: "follow", Content: "test"}
	if err := db.Create(&notification).Error; err != nil {
		t.Fatal(err)
	}
	request("POST", "/notification/list", map[string]int{}, bob.Token, 200)
	request("POST", "/notification/unreadCount", map[string]int{}, bob.Token, 200)
	request("POST", "/notification/markRead", map[string]uint{"id": notification.ID}, alice.Token, 200)
	if err := db.First(&notification, notification.ID).Error; err != nil {
		t.Fatal(err)
	}
	if notification.IsRead {
		t.Fatal("another account marked a notification read")
	}
	request("POST", "/notification/markRead", map[string]int{}, bob.Token, 200)
	count := decodeJSON[map[string]int](t, request("POST", "/notification/unreadCount", map[string]int{}, bob.Token, 200))
	if count["count"] != 0 {
		t.Fatal("notifications not marked read")
	}
	request("POST", "/social/unfollow", map[string]uint{"vlogger_id": alice.AccountID}, bob.Token, 200)
	request("POST", "/account/logout", map[string]int{}, bob.Token, 200)
	request("GET", "/account/me", nil, bob.Token, 401)
	request("GET", "/notification/stream?token="+bob.Token, nil, "", 401)
	request("POST", "/account/refresh", map[string]string{"refresh_token": bob.RefreshToken}, "", 401)
	request("POST", "/account/changePassword", map[string]string{"username": "alice2", "old_password": "password123", "new_password": "newpassword123"}, "", 200)
	request("GET", "/account/me", nil, alice.Token, 401)
	request("POST", "/account/login", credentials("alice2", "password123"), "", 401)
	loggedIn := decodeJSON[account.LoginResponse](t, request("POST", "/account/login", credentials("alice2", "newpassword123"), "", 200))
	expired := time.Now().Add(-time.Hour)
	if err := db.Model(&account.Account{}).Where("id = ?", alice.AccountID).Update("refresh_expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	request("POST", "/account/refresh", map[string]string{"refresh_token": loggedIn.RefreshToken}, "", 401)
}
