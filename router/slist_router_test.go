package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSelfHelpListRouter_BindsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	r := &SelfHelpListRouter{}
	if err := r.InitRouter(eng); err != nil {
		t.Fatalf("InitRouter returned err: %v", err)
	}

	cases := []struct {
		name string
		req  *http.Request
	}{
		{"query-form-get", httptest.NewRequest(http.MethodGet, "/slist?way=abc", nil)},
		{"query-form-post", httptest.NewRequest(http.MethodPost, "/slist?way=abc", nil)},
		{"query-form-options", httptest.NewRequest(http.MethodOptions, "/slist?way=abc", nil)},
		{"path-form-get", httptest.NewRequest(http.MethodGet, "/slist/abc", nil)},
		{"path-form-post", httptest.NewRequest(http.MethodPost, "/slist/abc", nil)},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, tc.req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d; want 200 (route bound for %s)",
				tc.name, w.Code, tc.req.Method)
		}
	}
}
