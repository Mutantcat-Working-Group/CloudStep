# 手动生效/失效地址 实现计划

> **给 Agent workers 的说明：** 用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐 task 执行。步骤用 `- [ ]` 复选框跟踪。

**目标：** 在 CloudStep 中实现 `POST /url/enable` 和 `POST /url/disable`，登录鉴权，按 id 改 `Url.Alive`；启用时复位 `Retry=0`，禁用时不动 `Retry`。顺手修 `GetPath` / `GetProxyPath`：跳过 `Alive=false`，全死返回 `""`。

**架构：** 一个事务 DAO `dao.UrlDao::UpdateUrlAlive(id, alive)`，DB 单行 CAS + 同步内存 cache `WorkCllection`。前端不在本轮范围内。

**技术栈：** Go、Gin v1.12、xorm v1.4、SQLite，无新依赖。

---

## 关键 xorm 细节（决定写法，必读）

xorm 默认的 `Update(&struct)` 只写 non-zero 字段。`false` 是 bool zero，写不进去。必须强制列：

- 启用：`session.Cols("alive","retry").ID(id).Update(&Url{Alive:true, Retry:0})` — 改 alive + retry
- 禁用：`session.Cols("alive").ID(id).Update(&Url{Alive:false})` — 只改 alive，retry 不动

两份 handler 都仅调 `dao.UpdateUrlAlive(id, alive)`，不直接触 xorm。

---

## Task 1: DAO failing tests (TDD red)

**Files:** Test: `dao/url_state_test.go` (新增).

- [ ] **Step 1: 写 `dao/url_state_test.go`**

```go
package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"testing"
)

// resetUrlCaches 重置三个 package 级 cache, 仿 /collection/slist_test.go::resetCaches.
func resetUrlCaches() {
	collection.MWorkCllection.Lock()
	collection.WorkCllection = make(map[string][]entity.Url)
	collection.MWorkCllection.Unlock()

	collection.MSelfHelpMode.Lock()
	collection.SelfHelpMode = make(map[string]entity.SelfHelp)
	collection.MSelfHelpMode.Unlock()

	collection.MProxyMode.Lock()
	collection.ProxyMode = make(map[string]entity.Proxy)
	collection.MProxyMode.Unlock()
}

func TestUpdateUrlAlive_Enable(t *testing.T) {
	resetUrlCaches()
	// seed: 一个 Alive=false 的 URL
	const (
		id   = 1
		coll = "coll"
		path = "http://u"
	)
	collection.MWorkCllection.Lock()
	collection.WorkCllection[coll] = []entity.Url{{Id: id, Parent: coll, Path: path, Alive: false, Retry: 3}}
	collection.MWorkCllection.Unlock()

	if !UpdateUrlAlive(id, true) {
		t.Fatalf("UpdateUrlAlive(enable) = false; want true")
	}

	// DB 校验
	var got entity.Url
	has, err := PublicEngine.ID(id).Get(&got)
	if err != nil || !has {
		t.Fatalf("DB fetch after enable: has=%v err=%v", has, err)
	}
	if !got.Alive || got.Retry != 0 {
		t.Fatalf("DB after enable: got Alive=%v Retry=%d; want true,0", got.Alive, got.Retry)
	}
	// cache 校验
	collection.MWorkCllection.Lock()
	cached := collection.WorkCllection[coll][0]
	collection.MWorkCllection.Unlock()
	if !cached.Alive || cached.Retry != 0 {
		t.Fatalf("cache after enable: got Alive=%v Retry=%d; want true,0", cached.Alive, cached.Retry)
	}
}

func TestUpdateUrlAlive_Disable(t *testing.T) {
	resetUrlCaches()
	const (
		id   = 2
		coll = "coll2"
		path = "http://v"
	)
	collection.MWorkCllection.Lock()
	collection.WorkCllection[coll] = []entity.Url{{Id: id, Parent: coll, Path: path, Alive: true, Retry: 7}}
	collection.MWorkCllection.Unlock()

	if !UpdateUrlAlive(id, false) {
		t.Fatalf("UpdateUrlAlive(disable) = false; want true")
	}

	var got entity.Url
	has, err := PublicEngine.ID(id).Get(&got)
	if err != nil || !has {
		t.Fatalf("DB fetch after disable: has=%v err=%v", has, err)
	}
	if got.Alive || got.Retry != 7 {
		t.Fatalf("DB after disable: got Alive=%v Retry=%d; want false,7", got.Alive, got.Retry)
	}
	collection.MWorkCllection.Lock()
	cached := collection.WorkCllection[coll][0]
	collection.MWorkCllection.Unlock()
	if cached.Alive || cached.Retry != 7 {
		t.Fatalf("cache after disable: got Alive=%v Retry=%d; want false,7", cached.Alive, cached.Retry)
	}
}

func TestUpdateUrlAlive_NotFound(t *testing.T) {
	resetUrlCaches()
	if UpdateUrlAlive(999999, true) {
		t.Fatalf("UpdateUrlAlive(nonexistent id) = true; want false")
	}
	if UpdateUrlAlive(999999, false) {
		t.Fatalf("UpdateUrlAlive(nonexistent id, false) = true; want false")
	}
}
```

注意：这份测试引用 `dao.UpdateUrlAlive` —— 尚未定义, 编译失败即 TDD red。

- [ ] **Step 2: 跑, 确认编译失败**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test -run 'TestUpdateUrlAlive' ./dao/ 2>&1 | head -30
```

Expected: compile error `undefined: dao.UpdateUrlAlive`. 若反现 PASS, 停并核对。

- [ ] **Step 3: 提交失败的测试**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && git add dao/url_state_test.go && git commit -m "test(dao): add UpdateUrlAlive failing tests (TDD red)"
```

---

## Task 2: 实现 `dao.UpdateUrlAlive` (TDD green)

**Files:** 修改: `dao/url_dao.go` (同文件已有 AddUrl / DeleteUrlById / UpdateUrlById).

- [ ] **Step 1: 在 `dao/url_dao.go` 末尾加 `UpdateUrlAlive`**

```go
// UpdateUrlAlive 按 id 设置 alive:
//   alive=true : alive=true 且 retry=0
//   alive=false: alive=false, retry 不动
// 双写 DB + cache。返回 false 当 id 不存在或事务错。
func UpdateUrlAlive(id int, alive bool) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	var err error
	if alive {
		_, err = session.Cols("alive", "retry").ID(id).Update(&entity.Url{Alive: true, Retry: 0})
	} else {
		_, err = session.Cols("alive").ID(id).Update(&entity.Url{Alive: false})
	}
	if err != nil {
		session.Rollback()
		return false
	}
	if affected, err := session.Commit(); err != nil || affected == 0 {
		session.Rollback()
		return false
	}

	// 同步 cache: 翻 WorkCllection 找同 Id 的 url
	MWorkCllection.Lock()
	for coll, urls := range WorkCllection {
		for i := range urls {
			if urls[i].Id == id {
				WorkCllection[coll][i].Alive = alive
				if alive {
					WorkCllection[coll][i].Retry = 0
				}
			}
		}
	}
	MWorkCllection.Unlock()

	return true
}
```

注意: xorm 中 `session.Commit()` 在某些版本返回 `(affected, err)`；若本地 xorm 1.4 与之不合, 改用 `err := session.Commit()` 并去掉 affected 检查。

- [ ] **Step 2: 跑 DAO 测试, 红变绿**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test -run 'TestUpdateUrlAlive' ./dao/ -v 2>&1 | tail -25
```

Expected: 三个 PASS + `ok  	com.mutantcat.cloud_step/dao`.

- [ ] **Step 3: 全量 build**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go build ./...
```
Expected: 无输出。

- [ ] **Step 4: 提交**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && git add dao/url_dao.go && git commit -m "feat(dao): add UpdateUrlAlive with DB+cache sync"
```

---

## Task 3: 给 GetPath/GetProxyPath 加 alive 过滤 (TDD green, 顺带修 bug)

**Files:**
- 修改: `collection/self_help_mode.go`
- 修改: `collection/proxy_mode.go`
- Test: `collection/url_state_read_test.go` (新增)

这是一个附带修复, 但给测试单独一个 task 以便单独验收。

- [ ] **Step 1: 先在 `collection/self_help_mode.go` 内补 `filterAlive`**

在文件末尾追加:

```go
// filterAlive 返回活地址切片; 全死返回空切片(非 nil, GetPath 据此返回 "").
func filterAlive(urls []entity.Url) []entity.Url {
	out := make([]entity.Url, 0, len(urls))
	for _, u := range urls {
		if u.Alive {
			out = append(out, u)
		}
	}
	return out
}
```

- [ ] **Step 2: 改 `GetPath` 用 `filterAlive`**

当前代码(已读)在 `case "random"` / `case "polling"` 分支里直接用 `WorkCllection[now.Point]` 选索引。修订: 在 `switch now.Mode` 之前先把 `WorkCllection[now.Point]` 拷出, 过滤后决定; 若空, return "";

替换为:

```go
WorkCllection.mu.Lock()
raw := WorkCllection[now.Point]
newIndex := now.Index
WorkCllection.mu.Unlock()
alive := filterAlive(raw)
if len(alive) == 0 {
	SelfHelpMode[way] = now
	return ""
}
switch now.Mode {
case "random":
	now.Index = util.RandInt(0, len(alive))
	return alive[now.Index].Path
case "polling":
	path := alive[now.Index%len(alive)].Path
	now.Index++
	if now.Index >= len(alive) {
		now.Index = 0
	}
	SelfHelpMode[way] = now
	return path
}
```

(伪代码: 实际写时按 xorm-mutex 命名/MSelfHelpMode/MWorkCllection 调整 — 必须能编译。先用 `MSelfHelpMode.Lock()` / `MWorkCllection.Lock()` 替换上面伪名。)

- [ ] **Step 3: 拷贝同一 helper 到 `collection/proxy_mode.go` 并改 `GetProxyPath`**

同上, 独立 helper, 改 GetProxyPath。

- [ ] **Step 4: 测试**

写 `collection/url_state_read_test.go` 用现成的 WorkCllection cache 注入 Alive=false 与 Alive=true 的混合 urls, 断言:

- `GetPath` / `GetProxyPath` 永远选到 Alive=true
- 全 Alive=false 时返回 ""

```go
package collection

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

func TestGetPath_SkipsDeadUrls(t *testing.T) {}
func TestGetPath_AllDeadReturnsEmpty(t *testing.T) {}
func TestGetProxyPath_SkipsDeadUrls(t *testing.T) {}
func TestGetProxyPath_AllDeadReturnsEmpty(t *testing.T) {}
```

(实现体按 spec §6 自拟 — 这里留作实现者任务, 不必黏到 plan。实现者用私有 cache 注入仿 /collection/slist_test.go。)

- [ ] **Step 5: 跑全量 collection + dao + router 包**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test ./collection/ ./dao/ ./router/ 2>&1 | tail -10
```

Expected: 全 pass。

- [ ] **Step 6: 提交**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep \
  && git add collection/self_help_mode.go collection/proxy_mode.go collection/url_state_read_test.go \
  && git commit -m "fix(collection): GetPath/GetProxyPath skip dead URLs"
```

(注: 实现者写 Task 3 测试时, 如果单纯把测试 function 留空会让 go test 全 PASS 但不 cover anything — 所以实现者必须把测试体实装。这是 Task 3 的核心。)

---

## Task 4: Router handlers `/url/enable` `/url/disable`

**Files:** 修改: `router/setting_router.go`.

- [ ] **Step 1: 在 `/url/delete` 后追加路由与 handler**

在 `InitRouter` 末尾追加:

```go
context.POST("/url/enable", LoginHandler(), enableUrl)
context.POST("/url/disable", LoginHandler(), disableUrl)
```

在文件底部加两个 handler:

```go
func enableUrl(c *gin.Context) {
	id, ok := bindId(c)
	if !ok {
		return
	}
	if dao.UpdateUrlAlive(id, true) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}

func disableUrl(c *gin.Context) {
	id, ok := bindId(c)
	if !ok {
		return
	}
	if dao.UpdateUrlAlive(id, false) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}

// bindId 从 JSON 取 id 并校验, 校验失败直接写 {"code":1} 并返回 false.
func bindId(c *gin.Context) (int, bool) {
	type body struct {
		Id int `json:"id"`
	}
	var b body
	if err := c.ShouldBindJSON(&b); err != nil || b.Id <= 0 {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return 0, false
	}
	return b.Id, true
}
```

- [ ] **Step 2: build**

```bash
go build ./...
```
Expected: 无输出。

- [ ] **Step 3: 跑 router 包**

```bash
go test ./router/ 2>&1 | tail -10
```

(无 handler 级别测试 — 现有 /router 没 handler 单测, 只靠 route-binding 验; 本次赌一次, 靠 smoke 测完整链路。)

- [ ] **Step 4: 提交**

```bash
git add router/setting_router.go && git commit -m "feat(router): add /url/enable and /url/disable"
```

---

## Task 5: Manual smoke (可选, 本地跑, 不提交)

**Files:** 无新文件。

- [ ] **Step 1: Boot server on :9300, 跑一轮 curl**

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep
go build -o /tmp/cs . && /tmp/cs 9300 &
SRV=$!
sleep 2

# 登录拿 cookie
curl -s -i -X POST http://127.0.0.1:9300/login -d '{"username":"admin96","password":"admin96"}' -o login.out
grep -i 'set-cookie' login.out || echo "NO-COOKIE"
COOKIE=$(grep -i 'set-cookie' login.out | sed 's/.*: //')

# 取某个已知 id (请先通过 /collection/geturls?id=<集合id> 确认一个 id)
# 例: id 1
echo "--- disable ---"
curl -s -X POST http://127.0.0.1:9300/url/disable -H "Cookie: $COOKIE" -d '{"id":1}'
echo "--- enable ---"
curl -s -X POST http://127.0.0.1:9300/url/enable -H "Cookie: $COOKIE" -d '{"id":1}'

# 用 sqlite3 CLI 验证 (管理者可手动):
#   sqlite3 ./cloud_step.db "SELECT id,parent,alive,retry FROM url LIMIT 10;"

kill $SRV 2>/dev/null
```

Expected: disable 响应 `{"code":0,...}` + enable 响应 `{"code":0,...}`；数据库可见 alive 字段反了又回。

- [ ] Step 2 (可选): 如果你手头 DB 中现存的集合 id 不确定, 先 `sqlite3 cloud_step.db ".tables"` + `SELECT id,name FROM collection;` 找到 id 后再验。

(此 smoke 不提交, 但建议至少跑一次。)

---

## Task 6: README checkbox 翻转

**Files:** 修改: `README.md`.

- [ ] **Step 1: `[ ]` 变 `[X]`**

在 README.md「六、开发进度」下, 把
```
- [ ] 手动生效或失效地址
```
替换成
```
- [X] 手动生效或失效地址
```

(注: 实现者确认一下是第 87 行, 不要撞别的 `[ ]`。)

- [ ] **Step 2: 提交**

```bash
git add README.md && git commit -m "docs: mark 'manual enable/disable' as completed in dev-progress"
```

---

## Self-Review 收尾清单

| 规格 task | commit 后应有 |
|---|---|
| F1 /url/enable / /url/disable | Task 4 末尾 PASS build |
| F2 id 校验 | Task 4 bindId |
| F3 启用 reset retry=0 | Task 2 TestUpdateUrlAlive_Enable DB + cache 校验 |
| F4 禁用 retry 不动 | Task 2 TestUpdateUrlAlive_Disable |
| F5 DB+cache 同步 | Task 2 UpdateUrlAlive 双写 |
| F6 不改 /url/add /delete /update UI | Task 4 只追加 |
| F7 修读取端 | Task 3 filterAlive |
| F8 心跳未来共用 | Task 2 单一入口 |
