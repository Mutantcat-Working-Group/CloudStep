# 空路径默认模式 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 「开发进度 · 空路径默认模式」

## 1. 背景

按 README 清单「开发进度 · 空路径默认模式」实现。

基础: `/self?way=...` 和 `/agent?way=...` 当前都要求必传 `way`, 缺失时直接返回 `{"code":404}`。然而在「未指定坐标」场景下, 用户/前端库通常期望系统有个兜底行为(比如返回某个"默认映射集"里的地址)。本次引入这个能力, 仅针对 `/self` 和 `/agent`: 两个前缀各自的默认映射集独立设置, 当 `way` 缺失时兜底返回; `way` 有值时完全不动。

## 2. 功能 / 非功能需求

### 功能需求

| # | 需求 |
|---|---|
| F1 | `GET /self` 不传 way= → 使用 `selfDefaultCollectionId` 对应的映射集 |
| F2 | `GET /agent` 不传 way= → 使用 `agentDefaultCollectionId` 对应的映射集 |
| F3 | `way=` 有值 → 走现有逻辑(**本次不做任何改动**, 保证契约不变) |
| F4 | `selfDefaultCollectionId=0` 或未配置 → 无兜底, 仍返回 `{"code":404}` |
| F5 | 默认映射集全部 URL 均为 `Alive=false`(全被 manual-disable 停掉) → 返回 404, 不返回死地址 |
| F6 | 新增的 default id 必须指向实际存在的 `collection` 表行; 不存在则拒绝写入 system_config |
| F7 | 配置项: 登录鉴权, 仅在管理后台可改 |
| F8 | 持久化: 重启后配置保留 |
| F9 | 后台"系统管理"页复用现有 `/sysconfig/*` 路由, 加两个下拉("自助默认集"/"代理默认集") |

### 非功能约束

| # | 约束 |
|---|---|
| C1 | 最小改动: 复用现有 `system_config` 表 + `SysConfig` 内存镜像 + 读写锁 |
| C2 | yml 新增 API: 无 — 复用 `/sysconfig/get` `/sysconfig/update` |
| C3 | 现有 `/self` `/agent` `/slist` 契约完全不变 |
| C4 | 启用时完全不破坏现有的 404 兜底行为 |
| C5 | 错误响应在 `code:1` 基础上加可选 `desc` 字段带出具体错误描述 |
| C6 | 默认集名走 DB 查(collection 表), 不引入新 cache |

## 3. API 契约

复用现有 `/sysconfig/get` + `/sysconfig/update`。

### 3.1 GET /sysconfig/get

字段扩两个:

```json
{
  "code": 0,
  "data": {
    "allowIntranetProxy": true,
    "selfDefaultCollectionId": 3,
    "agentDefaultCollectionId": 0
  }
}
```

未配时两个 id = 0(int zero value)。

### 3.2 POST /sysconfig/update

请求体接受全部已存在字段 + 新两个; 都可选(缺的不变):

```json
{
  "allowIntranetProxy": true,
  "selfDefaultCollectionId": 3,
  "agentDefaultCollectionId": 5
}
```

校验:
- `selfDefaultCollectionId > 0` → 必须在 `collection` 表存在, 否则返回 `{"code":1,"msg":"参数错误","desc":"selfDefaultCollectionId not found"}`
- `agentDefaultCollectionId > 0` → 同上
- `=0` → 视为"清除默认"(把 default 设回无)

成功返回 `{"code":0,"msg":"success"}`。

### 3.3 /self /agent 行为边界

| 条件 | 返回 |
|---|---|
| `selfDefaultCollectionId=0` 或 collection miss | `{"code":404}` |
| `selfDefaultCollectionId>0` 但集下 URL 全部 manual-disable | `{"code":404}` |
| `selfDefaultCollectionId>0` 且集下有 Alive=true URL | 按现有 random/polling 模式选出一条返回 |
| `way=` 有值 | **完全走现有 SelfHelpMode/ProxyMode 路径; 默认配置完全不读** |

`/agent` 同理, 对 `agentDefaultCollectionId`。

## 4. 代码位置

### 修改的文件 (4 个)

| 文件 | 改动 |
|---|---|
| `entity/dao_entity.go` | `SystemConfig` 新增 `SelfDefaultCollectionId int`、`AgentDefaultCollectionId int` 字段 |
| `dao/system_config_dao.go` | `GetSystemConfig()` 返回新字段; `UpdateSystemConfig(...)` 按 id>0 校验 collection 存在, 双写 DB + cache |
| `router/setting_router.go` | `updateSysConfig` handler 扩 json body struct, 校验两个 id; 缺则保持为 0 |
| `util/system_config.go` | `SysConfig` 内存镜像加两个字段, 读写逻辑同步, 新增 `GetSysConfigMirror()` 只读访问器(RLock 保护) |

### 一份新 helper

挂 `util/resolve_default.go` (新文件), 导出:

```go
// ResolveWayCollection 在 GetPath / GetProxyPath 找 way 失败后调用一次:
//
//   - way != "" → 返回 ("", false); 让现有路径自行处理
//   - way == "" 且 defaultId > 0 → 通过 dao.GetCollectionNameById 查得集合名; miss → ("", false)
//   - way == "" 且 defaultId == 0 → ("", false)
func ResolveWayCollection(way string, defaultId int) (collName string, ok bool)
```

- 走 DB 读 `collection` 表(按 id, 单行 pk 查询, 便宜), 无需新 cache
- 不持有 cache 锁

## 5. 数据流

```
GET /self?  (way 缺失)
  └─ selfhelpHandler
       └─ SelfHelpMode.GetPath("") 返回 0/false (way 不存在该 key)
           现有 handler 返 404
           但修订后 GetPath 内部:
             way = util.ResolveWayCollection(way, SysConfig.SelfDefaultCollectionId)
             if way == "" { return "" }
             way 再走现有 SelfHelpMode[way].Point → WorkCllection[point]
```

修订点: 在 `GetPath` 函数顶部加一段:

```go
func GetPath(way string) string {
    if way == "" {
        if resolved, ok := util.ResolveWayCollection(way, util.GetSysConfigMirror().SelfDefaultCollectionId); ok {
            way = resolved
        } else {
            return ""
        }
    }
    // ... 以下不变 ...
}
```

`util.GetSysConfigMirror()` 需要在 `util/system_config.go` 加一个安全的镜像访问器:

```go
func GetSysConfigMirror() SystemConfig {
    SystemConfigMu.RLock()
    defer SystemConfigMu.RUnlock()
    return SysConfig
}
```

同样改 `GetProxyPath`(用 `AgentDefaultCollectionId`) + `GetSelfHelpList`(本列表集不靠 way, 所以不需要)。

`/slist` 列表端点不受影响。

## 6. 错误处理 + 边界

| 场景 | 处理 |
|---|---|
| 老 DB 重启后新字段 NULL | xorm `Sync2(SystemConfig{})` 在 `orm_instance.go` 已注册; 线上 `alter table` 装新列默认 0; 0 = 未配, 兜底 404 ✅ |
| 管理员配的指向已被删的 collection | Sync2 启动时不校验; Read 时 `dao.GetCollectionNameById` miss → helper ok=false → 404; **不 panic** |
| default id 指的集全部 URL 被 manual-disable | `filterAlive` 后 len=0 → GetPath/GetProxyPath return "" → handler 返 404 ✅ |
| 客户端传 collection 在 DB 不存在的 id | `updateSysConfig` 校验 miss → `code:1, desc:...` ❌ |
| `way=` 有值但同时配了 default id | helper 完全不看 default id; 走现有路径; 默认配置透明不可见 ✅ |
| `/slist` 查映射集照_show_全部 URL | `/slist` 读 filterAlive 外, 不受影响 |
| 并发 `updateSysConfig` 写与 `GetPath` 读 | `SysConfigMu` 读写锁
#  (SYSConfig.mu) 在 `util/system_config.go`

## 7. 测试

### Dao 层

- `UpdateSystemConfig` 写入 (allowIntranetProxy, selfDefaultCollectionId, agentDefaultCollectionId) → DB 读取 + 内存镜像双面判
- 校验 miss: `selfDefaultCollectionId=999 → collection miss → 返回 false`

### 读端点修订

- `GetPath` 在 `way==""` 且 SysConfig.SelfDefaultCollectionId>0 + 映射集有活 url → 能返回地址
- `GetPath` 全活 url manual-disable 时返回 0/false → handler 返 code:404
- `GetPath` way 有值时完全不读 SysConfig.SelfDefaultCollectionId(即 default 配置对显式 way 完全透明)
- `GetProxyPath` 同上

### 回归

- `/sysconfig/get` 与已有 client(如 `/self` 业务的 getSysConfig 前端)不致字段破坏(`code` 旁扩充字段, 不影响老客户端)
- `/slist` 行为不变

## 8. 风险与备忘

- `desc` 字段是响应形状的小扩, 带 detail 错误描述。老客户端都读 `code`/`msg`, 加字段不会破坏它们, 但前端想做 i18n/error 展示时注意 `desc` 可选。
- 重启时 `Sync2(SystemConfig{})` 若已跑过(第一次装列后)不会重复改 schema(会 no-op)。无需 migration 脚本。
- 后台"系统管理"页(dropdown)需要前端单独改 layui 模板, 本次 spec 只管后端; 单独前端追做。
- 默认集全部 URL 全死场景的 404 兜底行为, 依赖 `collection.GetPath` 现有 `filterAlive` 已修(见本次同期的 `fix(collection): GetPath/GetProxyPath skip dead URLs`)。
- `util.ResolveWayCollection` 的 DB 读是 O(1) pk lookup, 无性能热点。未来若成热点再 cache。
- **`GetSysConfigMirror()` 是本次新增的 `util/system_config.go` RLock 只读访问器**。现有 `GetSystemConfig()` 走 dao 层开 DB 事务, 调用频率虽低但 GetPath 是热路径, 故镜像只读是 right call。

### 实施路径中的架构偏差 (import cycle)

spec §4/`util/resolve_default.go` 的计划写法是 `util` 直接调 `dao.GetCollectionNameById`( plan `implementation-plans/2026-07-12-empty-path-default-mode.md` §Task 3 原文)。**Go 拒绝这一写法**:`util → dao` 经 `collection_dao.go util import` 形成 `dao → collection → util → dao` 循环,compile error `import cycle not allowed`。

落地方式改为**resolver 注入**:

- `util/resolve_default.go` 暴露包级变量 `defaultCollectionResolver func(int) string`,由 `SetDefaultCollection Resolver(fn)` 注入。`ResolveWayCollection` 仅调 resolver,不 import dao。
- `dao/system_config_dao.go::InitSystemConfig` 在最后执行 `util.SetDefaultCollectionResolver(GetCollectionNameById)`。`InitSystemConfig` 由 `dao/orm_instance.go::init()` 启动时调用(在 ServeHTTP 之前),故首个 HTTP 请求到来时 resolver 已安装,**冷启动安全**。
- 单测路径:`util/resolve_default_test.go` 直接 inject mock resolver(无 DB / 无 cwd 依赖);`dao/system_config_dao.go::UpdateSystemConfig` 的 id 校验路径另有集成测试通过 `InitSystemConfig` 注入的真实 `GetCollectionNameById` 覆盖。
- 计划里的 `GetCollectionNameById` 调用语义 = `entity.Collection{ID}.Name`,通过 resolver 注入完全等价保留。

`ResolveWayCollection(way, defaultId)` 的公开签名、行为契约与 spec §5 数据流图完全一致;仅 internal resolver 是老计划为了绕开 import cycle 的"最佳等价"实现。
