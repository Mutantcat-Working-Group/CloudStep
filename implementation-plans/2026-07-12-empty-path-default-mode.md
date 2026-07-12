# 空路径默认模式 实现计划

> 给 agent workers 的说明:用 `superpowers:subagent-driven-development` 或 `superpowers:executing-plans` 逐 task 执行。步骤 `- [ ]` 复选框跟踪。

**目标:** 实现"空路径默认模式": 当 `/self` 或 `/agent` 不传 `way=` 时,使用 `system_config` 表里配置的默认映射集兜底。现有 `/self` `/agent` 契约完全不变(`way` 有值时绝不读配置)。

**架构:** 复用现有 `system_config` 表加两个 `int` 字段(`SelfDefaultCollectionId`, `AgentDefaultCollectionId`);管理后台 `/sysconfig/get` `/sysconfig/update` 只扩字段不改路由;新增 `util.ResolveWayCollection(way, defaultId)` 在 `GetPath` / `GetProxyPath` 内部按需调用;热路径通过新 `util.GetSysConfigMirror()` 读镜像,绕过 DB。

**技术栈:** Go / Gin / xorm / sqlite,无新依赖。

---

## 必读:真·代码 surface

这些文件是本次改动的锚点,实现者必须先读:

- `util/system_config.go` — **关键类型细节**:
  - util 包里现有类型是 **`systemConfig`(unexported)**,字段仅 `AllowIntranetProxy bool`。
  - 访问器 `SetAllowIntranetProxyallow bool)` / `AllowIntranet() bool`。
  - **没有** 现成的"Get full mirror"访问器。本次需要**新增** `GetSysConfigMirror() → entity.SystemConfig`,通过 `RLock` 返回DB复本(参见 `dao.GetSystemConfig()` 已做 DB 读)。
  - 注意: `util.systemConfig`(unexported)与 `entity.SystemConfig`(exported,含 `Id` / `AllowIntranetProxy`)是**两种不同类型**。新增字段要**两边都加**: `entity.SystemConfig` 加 `SelfDefaultCollectionId / AgentDefaultCollectionId int`;util 的镜像类型视实现选其一:
    - 方案 A(推荐): 抛弃 util 的 `systemConfig`, 改用 `util` 包镜像持有 `entity.SystemConfig`, 现有 `sysCfg`/`SetAllowIntranetProxy`/`AllowIntranet` 改为 `cfg`/`SetField`/`GetSysConfigMirror`;这样 *router 调用 `dao` 拿 `*entity.SystemConfig`* 与 `util.GetSysConfigMirror()→entity.SystemConfig` 类型一致。
    - 方案 B: util 的 `systemConfig` 加 `SelfDefaultCollectionId/AgentDefaultCollectionId` 字段, 再提供 `GetSysConfigMirror()→entity.SystemConfig` 在内部构造。类型对齐多一层 cast。
    - **本计划按方案 A, 不混两种类型。**

- `entity/dao_entity.go` `SystemConfig` — 当前字段 `Id / AllowIntranetProxy`。本次加 `SelfDefaultCollectionId int` / `AgentDefaultCollectionId int`(各 `json:"selfDefaultCollectionId" json:"agentDefaultCollectionId"`)。

- `dao/system_config_dao.go::UpdateSystemConfig` — 现阶段 `session.ID(systemConfigId).Update(&in)` 这种写法会把 `in` 里的 zero-value `int=0` 写进 DB(因为 int 的 zero 是 0,硬写), 所以当管理员把 default id 从 N 改回 0(清除默认), 原本写法会写成 0; 但**管理员想"把 AllowIntranetProxy 保留原值"时**,本次调 `UpdateSystemConfig(entity.SystemConfig{SelfDefaultCollectionId:3})` 会把 `AllowIntranetProxy` 写成 `false`(zero) —— 覆盖了老配置。所以本次必须:
  - 把 `UpdateSystemConfig` 的 DB 写改为 `Cols("allow_intranet_proxy").Update(...)` + 按 `in.SelfDefaultCollectionId > 0`  加 `"self_default_collection_id"` 到强制列, `==0` 不加(但 0 是 reset to 0, 不能 clear — 见下方)。
  - 更好的方式: 调用方显式传整个 `entity.SystemConfig`, UpdateSystemConfig **只写调用方 override 过的字段**,用 `dao.GetSystemConfig()` 读旧值, 合并后 全量写 **但指定强制列**。
  - 最终实施采用:**合并**:先 `old := GetSystemConfig()`; 再把 `in` 的非-zero 字段覆盖上去; 调用 `session.Cols(<所有非zero字段>).Update(&merged)`。
  - 但 `AllowIntranetProxy` 语义: false 在 Go 里是 zero, 管理员显式传 false 会被当"缺省"。规避法: 调用方前端永远传完整三个字段, 后端不做 zero-skip, merged = in (前端全量)。**本计划采用"前端全量,后端全量"**:handler 把 JSON 解析成完整 `entity.SystemConfig` 后, 组装合并(默认 fallback 到 原值), 调 `UpdateSystemConfig` 直接全量; UpdateSystemConfig 内 `Cols(...)` 显式列所有三个字段。

- `dao/collection_dao.go:108` `GetCollectionNameById(id int) string` — 现成的, resolver 不掉 cache。

- `router/setting_router.go:57` `updateSysConfig` — 当前 `body struct { AllowIntranetProxy bool }`, 扩两个 `int`; 加 id>0 经 `dao.GetCollectionNameById` 校验; 校验失败返 `code:1, desc:...`。

- `router/setting_router.go:49` `getSysConfig` — 直透 `dao.GetSystemConfig()`, 自动含新字段。

- `collection/self_help_mode.go::GetPath` — 顶部加:
  ```go
  if way == "" {
      if resolved, ok := util.ResolveWayCollection(way, util.GetSysConfigMirror().SelfDefaultCollectionId); ok {
          way = resolved
      } else {
          return ""
      }
  }
  ```
- `collection/proxy_mode.go::GetProxyPath` — 同上但读 `AgentDefaultCollectionId`。

- `collection/slist.go::GetSelfHelpList` — 不动。

---

## Task 1: 给 `entity.SystemConfig` + util 镜像加字段 + TDD failing tests

**Files:**
- 改 `entity/dao_entity.go`
- 改 `util/system_config.go`(方案 A: 退役 `systemConfig`, 改用 `entity.SystemConfig` 镜像)
- 测试: `util/system_config_test.go`(新增)

- [ ] Step 1: 改 `entity/dao_entity.go::SystemConfig`:

```go
type SystemConfig struct {
	Id int `xorm:"pk" json:"id"`

	// AllowIntranetProxy 是否允许反代理目标为私有/内网/回环/链路本地地址。
	AllowIntranetProxy bool `xorm:"notnull" json:"allowIntranetProxy"`

	// SelfDefaultCollectionId / AgentDefaultCollectionId 配置缺失 way= 时的
	// 自助 / 代理模式默认映射集; 0 表示未配置。
	SelfDefaultCollectionId  int `xorm:"notnull" json:"selfDefaultCollectionId"`
	AgentDefaultCollectionId int `xorm:"notnull" json:"agentDefaultCollectionId"`
}
```

- [ ] Step 2: 整段替换 `util/system_config.go`(方案 A):

```go
package util

import (
	"com.mutantcat.cloud_step/entity"
	"sync"
)

// sysCfg 是 entity.SystemConfig 的内存镜像,热路径读取绕过 DB。
var (
	sysCfg   = entity.SystemConfig{AllowIntranetProxy: true}
	sysCfgMu sync.RWMutex
)

// SetSystemConfigFromDao 由 InitSystemConfig 与 updateSysConfig 成功后调用,
// 把最新 DB 值整块替换进镜像。
func SetSystemConfigFromDao(c entity.SystemConfig) {
	sysCfgMu.Lock()
	defer sysCfgMu.Unlock()
	sysCfg = c
}

// GetSysConfigMirror 只读返回当前镜像(RLock 保护)。热路径用。
func GetSysConfigMirror() entity.SystemConfig {
	sysCfgMu.RLock()
	defer sysCfgMu.RUnlock()
	return sysCfg
}

// AllowIntranet 是否允许代理到内网(向后兼容入口)。
func AllowIntranet() bool {
	return GetSysConfigMirror().AllowIntranetProxy
}
```

向后兼容: `SetAllowIntranetProxy` 被 `proxy_util.go`/`ip.go` 等地引用者多处。处理: 检索全部调用点,都改成 `SetSystemConfigFromDao(entity.SystemConfig{AllowIntranetProxy: x})` — 唯独会覆盖另外两个 default id,所以更稳是:无调用点 → 直接删; 有调用点 → 改成 `c := GetSysConfigMirror(); c.AllowIntranetProxy = x; SetSystemConfigFromDao(c)`。实现者 grep 后决定。

- [ ] Step 3: 写 `util/system_config_test.go`(TDD red):

```go
package util

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

func TestGetSysConfigMirror_InitDefault(t *testing.T) {
	before := sysCfg
	defer func() { sysCfg = before }()

	SetSystemConfigFromDao(entity.SystemConfig{AllowIntranetProxy: true, SelfDefaultCollectionId: 3, AgentDefaultCollectionId: 5})
	got := GetSysConfigMirror()

	if got.AllowIntranetProxy != true || got.SelfDefaultCollectionId != 3 || got.AgentDefaultCollectionId != 5 {
		t.Fatalf("mirror = %+v; want {AllowIntranetProxy:true Self:3 Agent:5}", got)
	}
	if !AllowIntranet() {
		t.Fatalf("AllowIntranet() = false; want true")
	}
}
```

Expected: 编译失败(`SelfDefaultCollectionId` 未定义于 `entity.SystemConfig`) → TDD red。

- [ ] Step 4: 跑确认 red

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test -run 'TestGetSysConfigMirror' ./util/ 2>&1 | head -25
```

- [ ] Step 5: 提交

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && git add entity/dao_entity.go util/system_config.go util/system_config_test.go && git commit -m "test(util): add GetSysConfigMirror failing tests + expand SystemConfig"
```

---

## Task 2: 实现 `InitSystemConfig` 同步 + `dao.GetSystemConfig/UpdateSystemConfig` 全字段

**Files:** 改 `dao/system_config_dao.go`。

- [ ] Step 1: 重写 `InitSystemConfig` 同步 3 个字段:

```go
func InitSystemConfig() {
	cfg := entity.SystemConfig{}
	have, err := PublicEngine.ID(systemConfigId).Get(&cfg)
	if err != nil {
		log.Fatal("读取系统配置失败: ", err)
	}
	if !have {
		cfg = entity.SystemConfig{Id: systemConfigId, AllowIntranetProxy: true}
		if _, err := PublicEngine.Insert(&cfg); err != nil {
			log.Fatal("写入默认系统配置失败: ", err)
		}
	}
	util.SetSystemConfigFromDao(cfg)
}
```

- [ ] Step 2: 改 `UpdateSystemConfig` 校验 id>0 + 全量写明列:

```go
func UpdateSystemConfig(in entity.SystemConfig) bool {
	// 校验 default id 指向存在的 collection
	if in.SelfDefaultCollectionId > 0 && GetCollectionNameById(in.SelfDefaultCollectionId) == "" {
		return false
	}
	if in.AgentDefaultCollectionId > 0 && GetCollectionNameById(in.AgentDefaultCollectionId) == "" {
		return false
	}

	in.Id = systemConfigId
	session := PublicEngine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return false
	}
	if _, err := session.Cols("allow_intranet_proxy", "self_default_collection_id", "agent_default_collection_id").ID(systemConfigId).Update(&in); err != nil {
		session.Rollback()
		return false
	}
	if err := session.Commit(); err != nil {
		session.Rollback()
		return false
	}
	util.SetSystemConfigFromDao(in)
	return true
}
```

注:xorm struct tag `xorm:"notnull"` + snake_case 命名 → DB 列名应为 `allow_intranet_proxy`/`self_default_collection_id`/`agent_default_collection_id`。实装时用 `go test ./dao/` 试跑, 看 xorm 报不报 column name mismatch; 如有, 按 xorm 期望名调整。

`GetCollectionNameById` 位于 `dao/collection_dao.go`, 同 package 内可直接调用。

- [ ] Step 3: util 测试 green

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test -run 'TestGetSysConfigMirror' ./util/ -v 2>&1 | tail -10
```

- [ ] Step 4: 跑全量 + build

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go build ./... && go test ./util/ ./dao/ 2>&1 | tail -15
```

Expected: 全 PASS。

- [ ] Step 5: 提交

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && git add dao/system_config_dao.go && git commit -m "feat(dao): UpdateSystemConfig validates default ids + sync mirror"
```

---

## Task 3: 新增 `util.ResolveWayCollection`(纯函数,TDD)

**Files:**
- 新 `util/resolve_default.go`
- 测试 `util/resolve_default_test.go`

- [ ] Step 1: 写测试 (`util/resolve_default_test.go`):

```go
package util

import (
	"com.mutantcat.cloud_step/dao"
	"testing"
)

func TestResolveWayCollection(t *testing.T) {
	// way 非空 → 不读 default, 返回 ("", false)
	if coll, ok := ResolveWayCollection("explicitWay", 3); coll != "" || ok {
		t.Fatalf("explicit way should bypass default; got (%q,%v)", coll, ok)
	}
	// defaultId==0 → ("", false)
	if coll, ok := ResolveWayCollection("", 0); coll != "" || ok {
		t.Fatalf("zero id should return miss; got (%q,%v)", coll, ok)
	}
	// defaultId 指向 miss 的 collection → ("", false)
	if coll, ok := ResolveWayCollection("", 999999); coll != "" || ok {
		t.Fatalf("nonexistent collection should miss; got (%q,%v)", coll, ok)
	}
	// defaultId 指向存在的 collection → (name, true)
	// 找 dev DB 里真实存在的 collection id
	var anyCollName string
	{
		var c entity.Collection
		has, _ := PublicEngine.Table("collection").Where("id = 1").Get(&c)
		if has {
			anyCollName = c.Name
		}
	}
	if anyCollName == "" {
		t.Skip("dev DB 没有 id=1 的 collection, 跳过正流")
	}
	if coll, ok := ResolveWayCollection("", 1); !ok || coll != anyCollName {
		t.Fatalf("existing default coll %q: got (%q,%v)", anyCollName, coll, ok)
	}
}
```

注:此测试依赖 dev DB(`./cloud_step.db` relative to cwd), 从 `dao/` 跑会读到 `dao/cloud_step.db`, 而从 repo root 跑读到根的。实现者**从 repo root 跑** (`cd .. && go test ./util/`), 避免测试 DB 漂移。`entity.Collection` 需 import。

- [ ] Step 2: 实现 `util/resolve_default.go`(TDD green):

```go
package util

import (
	"com.mutantcat.cloud_step/dao"
)

// ResolveWayCollection 在 /self /agent 找 way 失败后调用一次,把缺失 way 落到
// 指定的默认映射集名称。
//
//   - way != "" → ("", false): 让现有路径自行处理
//   - defaultId == 0 → ("", false): 未配
//   - defaultId > 0 但 collection miss → ("", false): 误配
//   - defaultId > 0 且存在 → (collectionName, true)
//
// 走 DB 读 collection 表(单行 pk 查), 不依赖 cache; 若未来成热点再 cache。
func ResolveWayCollection(way string, defaultId int) (collName string, ok bool) {
	if way != "" || defaultId <= 0 {
		return "", false
	}
	name := dao.GetCollectionNameById(defaultId)
	if name == "" {
		return "", false
	}
	return name, true
}
```

- [ ] Step 3: 回 repo root 跑 util 测试

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test ./util/ -v 2>&1 | tail -20
```

Expected: `PASS`。

- [ ] Step 4: 提交

```bash
git add util/resolve_default.go util/resolve_default_test.go && git commit -m "feat(util): add ResolveWayCollection helper"
```

---

## Task 4: 给 `GetPath` / `GetProxyPath` 加默认分支

**Files:**
- 改 `collection/self_help_mode.go`(顶部)
- 改 `collection/proxy_mode.go`(顶部)

- [ ] Step 1: 在 `GetPath` 顶部加(wrap 现有 whole-way 路径):

```go
func GetPath(way string) string {
	// 默认配置兜底: way 缺失时, 使用 system_config 里配置的默认映射集名称替换 way。
	if way == "" {
		if resolved, ok := util.ResolveWayCollection(way, util.GetSysConfigMirror().SelfDefaultCollectionId); ok {
			way = resolved
		} else {
			return ""
		}
	}
	// ... 以下原封不改 ...
}
```

修改位置参考 `collection/self_help_mode.go:4`, 在 `MWorkCllection.Lock()` 之前插入这一段。

- [ ] Step 2: 同 `GetProxyPath` 在 `collection/proxy_mode.go`, 读 `AgentDefaultCollectionId`。

- [ ] Step 3: 跑 collection / util / dao 包

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test ./collection/ ./util/ ./dao/ 2>&1 | tail -15
```

Expected: PASS(原 `/self` `/agent` 链路的契约回归: way 有值完全不读 sysconfig)。

- [ ] Step 4: 提交

```bash
git add collection/self_help_mode.go collection/proxy_mode.go && git commit -m "feat(collection): GetPath/GetProxyPath use default collection on empty way"
```

---

## Task 5: 扩 `router/setting_router.go::updateSysConfig` 校验 default id

**Files:** 改: `router/setting_router.go`。

- [ ] Step 1: 在 `updateSysConfig` 前面加 id 校验层 + body 扩字段:

```go
func updateSysConfig(c *gin.Context) {
	type body struct {
		AllowIntranetProxy      bool `json:"allowIntranetProxy"`
		SelfDefaultCollectionId  int  `json:"selfDefaultCollectionId"`
		AgentDefaultCollectionId int  `json:"agentDefaultCollectionId"`
	}
	var b body
	if c.ShouldBindJSON(&b) != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	// 校验 default id 指向存在的 collection (0 表示"清除默认", 合法)
	if b.SelfDefaultCollectionId > 0 && dao.GetCollectionNameById(b.SelfDefaultCollectionId) == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误", "desc": "selfDefaultCollectionId not found"})
		return
	}
	if b.AgentDefaultCollectionId > 0 && dao.GetCollectionNameById(b.AgentDefaultCollectionId) == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误", "desc": "agentDefaultCollectionId not found"})
		return
	}
	if dao.UpdateSystemConfig(entity.SystemConfig{
		AllowIntranetProxy:      b.AllowIntranetProxy,
		SelfDefaultCollectionId:  b.SelfDefaultCollectionId,
		AgentDefaultCollectionId: b.AgentDefaultCollectionId,
	}) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}
```

- [ ] Step 2: build

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go build ./...
```

Expected: 无输出。(编译时如遇到 `entity.SystemConfig{}` 与 `b.AllowIntranetProxy` 不同类型冲突, 核对字段类型。)

- [ ] Step 3: 跑全量

```bash
cd /Volumes/Old_Solidity/Projects/Mutantcat/CloudStep && go test ./... 2>&1 | tail -15
```

Expected: 全 PASS。

- [ ] Step 4: 提交

```bash
git add router/setting_router.go && git commit -m "feat(router): updateSysConfig validates default collection ids"
```

---

## Task 6: README `[ ]` → `[X]` + 可选 smoke

**Files:** 改: `README.md`。

- [ ] Step 1: `[ ]` 变 `[X]`:

把:
```
- [ ] 空路径默认模式
```
换成:
```
- [X] 空路径默认模式
```

- [ ] Step 2: 可选 smoke(二进制启动 + curl `/self?`(no way), 已配了 default collection 的 id 时返回 200; 没有配时返回 404)。(smoke 方法参考本次同期 Task 5 of 手动启停)。

- [ ] Step 3: 提交 README

```bash
git add README.md && git commit -m "docs: mark 'empty-path default mode' as completed in dev-progress"
```

---

## Self-Review

| spec req | Task 实现 |
|---|---|
| F1 空 way /self → 默认集 | Task 4 self_help_mode |
| F2 空 way /agent → 默认集 | Task 4 proxy_mode |
| F3 way 有值完全不动 | Task 4 的 if way=="" 分支护住,Task 0 (entity) |
| F4 id=0 未配 → 404 | Task 3 resolve 的 defaultId<=0 返 false |
| F5 默认集全死 → 404 | Task 4 后段 filterAlive 过滤后 len=0, return ""(同期已修) |
| F6 id miss 拒绝写入 | Task 1 entity field + Task 2 UpdateSystemConfig 校验 |
| F7 登录鉴权 | Task 5 复用 LoginHandler |
| F8 持久化 | Task 1 entity + Task 2 UpdateSystemConfig DB 写 |
| F9 UI 复用 system_config | Task 5 |

**同步 surface 往返:** Task 1 util 测试时若 `entity.SystemConfig` 结构 tag 与 DB 列名冲突(go test 报 `redirector`), 按 xorm 期望列名调一次 entity struct tag。累计到 Task 5 smoke 时一并验。
