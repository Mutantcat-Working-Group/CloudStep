## Task 3: Route-binding test (TDD red)

**Files:** Test `router/slist_router_test.go` only. Production router lands in Task 4.

- [ ] **Step 1: Write the route-binding test**

```go
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
		{"query-form-get",     httptest.NewRequest(http.MethodGet,    "/slist?way=abc", nil)},
		{"query-form-post",    httptest.NewRequest(http.MethodPost,   "/slist?way=abc", nil)},
		{"query-form-options", httptest.NewRequest(http.MethodOptions,"/slist?way=abc", nil)},
		{"path-form-get",      httptest.NewRequest(http.MethodGet,    "/slist/abc",    nil)},
		{"path-form-post",     httptest.NewRequest(http.MethodPost,   "/slist/abc",    nil)},
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
```

Path-form subs (`/slist/abc`) use gin catch-all `/slist/*name`. `util.GetWayParam` reads `name=abc` from the path param when no `?way=` query is present (its precedence query > form > json > path handles both forms).

References `SelfHelpListRouter` - undefined on purpose. This is the deliberate TDD red for Task 4.

- [ ] **Step 2: Run, verify compile failure**

```bash
go test -run 'TestSelfHelpListRouter_BindsRoutes' ./router/ 2>&1 | head -20
```
Expected: compile error mentioning undefined `SelfHelpListRouter`.

- [ ] **Step 3: Commit**

```bash
git add router/slist_router_test.go && git commit -m "test(router): add SelfHelpListRouter binding test (TDD red)"
```

---

## Task 4: Implement `SelfHelpListRouter` and register (TDD green)

**Files:** Create `router/slist_router.go`, modify `main.go`.

- [ ] **Step 1: Create `router/slist_router.go`**

```go
package router

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
)

// SelfHelpListRouter exposes GET /slist?way=... (and /slist/:way), a public
// read of one mapping set's URL list resolved by `way`. No auth - matches
// /self posture.
type SelfHelpListRouter struct{}

func (router *SelfHelpListRouter) PrepareRouter() error { return nil }

func (router *SelfHelpListRouter) InitRouter(context *gin.Engine) error {
	context.Any("/slist", slist)
	context.Any("/slist/*name", slist)
	return nil
}

func (router *SelfHelpListRouter) DestroyRouter() error { return nil }

func slist(c *gin.Context) {
	way := util.GetWayParam(c)
	if way == "" {
		c.JSON(200, gin.H{"code": 404})
		return
	}

	source, point, urls := collection.GetSelfHelpList(way)
	if source == collection.SrcNone {
		c.JSON(200, gin.H{"code": 404})
		return
	}

	c.JSON(200, gin.H{
		"code":  0,
		"msg":   "success",
		"way":   way,
		"mode":  source.String(),
		"point": point,
		"data":  urls,
	})
}
```

- [ ] **Step 2: Register the router in `main.go`**

Change:
```go
	lifecycle.RegisterRouter(gin, &router.WebRouter{},
		&router.LoginRouter{},
		&router.SelfHelpRouter{},
		&router.ProxyRouter{},
		&router.PingRouter{},
		&router.SettingRouter{},
	)
```
To (append last argument):
```go
	lifecycle.RegisterRouter(gin, &router.WebRouter{},
		&router.LoginRouter{},
		&router.SelfHelpRouter{},
		&router.ProxyRouter{},
		&router.PingRouter{},
		&router.SettingRouter{},
		&router.SelfHelpListRouter{},
	)
```

- [ ] **Step 3: Build the whole module**

```bash
go build ./...
```
Expected: no output.

- [ ] **Step 4: Run router tests (now green)**

```bash
go test -run 'TestSelfHelpListRouter_BindsRoutes' ./router/ -v 2>&1 | tail -10
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add router/slist_router.go main.go && git commit -m "feat(router): add /slist self-help list endpoint"
```

---

## Task 5: HTTP smoke test (on-demand, build-tag `smoke`)

**Files:** `slist_smoke_test.go` at the repo root. Excluded from default `go test ./...` via the `smoke` buid tag. Runs the real process.

- [ ] **Step 1: Write the smoke harness**

```go
//go:build smoke
// +build smoke

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

// Manual smoke harness:
//   go test -tags=smoke -run TestSlistSmoke -v .
//
// Without the `smoke` tag this file is excluded from normal runs.
func TestSlistSmoke(t *testing.T) {
	_ = exec.Command("pkill", "-f", "cloud_step").Run()
	time.Sleep(200 * time.Millisecond)

	bin := "/tmp/cloud_step.smoke"
	build := exec.Command("go", "build", "-o", bin, "./...")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	srv := exec.Command(bin, "9192")
	if err := srv.Start(); err != nil {
		t.Fatalf("server start failed: %v", err)
	}
	defer func() { _ = srv.Process.Kill() }()

	base := "http://127.0.0.1:9192"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(10 * time.Second)
	var up bool
	for time.Now().Before(deadline) {
		if r, err := client.Get(base + "/check"); err == nil {
			r.Body.Close()
			up = true
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !up {
		t.Fatal("server did not come up on :9192")
	}

	checkEnvelope := func(name, url string) {
		resp, err := client.Get(url)
		if err != nil {
			t.Errorf("%s: GET %s: %v", name, url, err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status = %d; want 200; body=%s", name, resp.StatusCode, body)
			return
		}
		var env struct {
			Code int              `json:"code"`
			Msg  string           `json:"msg"`
			Way  string           `json:"way"`
			Mode string           `json:"mode"`
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &env); err != nil {
			t.Errorf("%s: invalid JSON: %s", name, body)
			return
		}
		fmt.Printf("%s -> code=%d way=%q mode=%q entries=%d msg=%q\n",
			name, env.Code, env.Way, env.Mode, len(env.Data), env.Msg)
	}

	// What this really exercises: server boots under the routed binary,
	// HTTP reaches /slist under GET + path-form, and the JSON envelope
	// parses cleanly for both the 200-shape (known way) and the 404-shape
	// (unknown / missing way) responses.
	checkEnvelope("unknown-way",       base+"/slist?way=does-not-exist-zzz")
	checkEnvelope("missing-way",       base+"/slist")
	checkEnvelope("path-form-unknown", base+"/slist/does-not-exist-zzz")
}
```

- [ ] **Step 2: Run the smoke harness**

```bash
go test -tags=smoke -run TestSlistSmoke -v . 2>&1 | tail -15
```

Expected: three printed lines of the form `... -> code=404 way="" mode="" entries=0 msg=""` for the 404-shape cases (HTTP 200 with `code:404` in the JSON envelope). If your `./cloud_step.db` has a known self-help way, drop in one extra `checkEnvelope("known-way", base+"/slist?way=<that-way>")` and assert `code=0 mode="selfhelp" entries>=1`.

- [ ] **Step 3: (optional) Commit the harness**

```bash
git add slist_smoke_test.go && git commit -m "test(smoke): add /slist on-demand process smoke harness"
```
Skip if you prefer keeping this file local-only.

---

## Task 6: Update README dev-progress checklist

**Files:** `README.md`

- [ ] **Step 1: Flip the `/slist` checkbox**

Line 80 in `README.md`:
```
- [ ] 自助列表（/slist）
```
becomes:
```
- [X] 自助列表（/slist）
```

- [ ] **Step 2: Commit**

```bash
git add README.md && git commit -m "docs: mark /slist as completed in dev-progress checklist"
```
