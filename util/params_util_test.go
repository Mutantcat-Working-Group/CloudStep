package util

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetWayParam_QueryForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.GET("/x", func(c *gin.Context) { c.String(200, GetWayParam(c)) })

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x?way=alpha", nil))
	if w.Code != 200 {
		t.Fatalf("query-form: status=%d want 200", w.Code)
	}
	if w.Body.String() != "alpha" {
		t.Fatalf("query-form: body=%q want alpha", w.Body.String())
	}
}

func TestGetWayParam_PathForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.GET("/self/:way", func(c *gin.Context) { c.String(200, GetWayParam(c)) })

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/self/gamma", nil))
	if w.Code != 200 {
		t.Fatalf("path-form: status=%d want 200 body=%q", w.Code, w.Body.String())
	}
	if w.Body.String() != "gamma" {
		t.Fatalf("path-form: body=%q want gamma", w.Body.String())
	}
}

func TestGetWayParam_JSONBodyPOST(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.POST("/p", func(c *gin.Context) { c.String(200, GetWayParam(c)) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/p", strings.NewReader(`{"way":"beta"}`))
	req.Header.Set("Content-Type", "application/json")
	eng.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("json: status=%d want 200 body=%q", w.Code, w.Body.String())
	}
	if w.Body.String() != "beta" {
		t.Fatalf("json: body=%q want beta", w.Body.String())
	}
}

func TestGetWayParam_GetEmptyBodyDoesNot400(t *testing.T) {
	// regression test: the pre-fix version 100% 400s on GET with empty body.
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.GET("/agent", func(c *gin.Context) { c.String(200, GetWayParam(c)) })

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agent", nil))
	if w.Code != 200 {
		t.Fatalf("get-empty: status=%d want 200 body=%q — this is the pre-existing 400 bug", w.Code, w.Body.String())
	}
}

func TestGetWayParam_PreferQueryOverPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.GET("/x/:way", func(c *gin.Context) { c.String(200, GetWayParam(c)) })

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x/pathway?way=qway", nil))
	if w.Body.String() != "qway" {
		t.Fatalf("prefer-query: body=%q want qway", w.Body.String())
	}
}

var _ = httptest.NewRecorder
