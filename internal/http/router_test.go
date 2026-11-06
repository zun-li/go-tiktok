package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReferenceRoutes(t *testing.T) {
	router := SetRouter(nil, nil, nil)
	actual := map[string]bool{}
	for _, route := range router.Routes() {
		actual[route.Method+" "+route.Path] = true
	}
	expected := []string{
		"GET /account/me",
		"GET /healthz",
		"GET /notification/stream",
		"POST /account/changePassword",
		"POST /account/findByID",
		"POST /account/findByUsername",
		"POST /account/getProfile",
		"POST /account/login",
		"POST /account/logout",
		"POST /account/refresh",
		"POST /account/register",
		"POST /account/rename",
		"POST /account/updateProfile",
		"POST /account/uploadAvatar",
		"POST /comment/delete",
		"POST /comment/listAll",
		"POST /comment/publish",
		"POST /feed/listByFollowing",
		"POST /feed/listByPopularity",
		"POST /feed/listByTag",
		"POST /feed/listLatest",
		"POST /feed/listLikesCount",
		"POST /like/isLiked",
		"POST /like/like",
		"POST /like/listMyLikedVideos",
		"POST /like/unlike",
		"POST /message/list",
		"POST /message/send",
		"POST /notification/list",
		"POST /notification/markRead",
		"POST /notification/unreadCount",
		"POST /social/follow",
		"POST /social/getAllFollowers",
		"POST /social/getAllVloggers",
		"POST /social/getCounts",
		"POST /social/unfollow",
		"POST /video/chunk/complete",
		"POST /video/chunk/init",
		"POST /video/chunk/status",
		"POST /video/chunk/upload",
		"POST /video/getDetail",
		"POST /video/listByAuthorID",
		"POST /video/publish",
		"POST /video/uploadCover",
		"POST /video/uploadVideo",
	}
	for _, route := range expected {
		if !actual[route] {
			t.Errorf("missing reference route: %s", route)
		}
	}
	for _, route := range []struct{ method, path string }{
		{"GET", "/account/me"}, {"POST", "/account/logout"}, {"POST", "/video/publish"},
		{"POST", "/video/chunk/init"}, {"POST", "/like/like"}, {"POST", "/comment/publish"},
		{"POST", "/social/follow"}, {"POST", "/feed/listByFollowing"}, {"POST", "/message/send"},
		{"GET", "/notification/stream"}, {"POST", "/notification/list"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(route.method, route.path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s returned %d without authentication", route.path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("health check: %d", response.Code)
	}
}
