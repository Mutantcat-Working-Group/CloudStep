# 自助列表 `/slist` 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README「开发进度 · 自助清单 `/slist`」

## 1. 背景与目标

「云阶」项目 README 的「六、开发进度」清单中,自助列表 `/slist` 标记为未完成。

目前已有的相关接口:

- `/self(/{way})` 客户端经 `way` 取**单个**地址(random/polling 模式)。
- `/agent` 走反向代理。
- 管理后台 `GET /collection/geturls?id=...` 用映射集 id 取该集 URL 列表,**需登录**。

缺失的能力:客户端以公开方式,通过 `way` 拿到**当前可用地址的完整列表**(含健康信息),由客户端自己做负载均衡决策——与「自助模式」的自助精神一致。

`/slist` 补齐这一环,并把口径从「映射集 id」提升为「`way` 坐标」(与管理后台解耦,与客户端用法一致)。

## 2. 需求与约束

### 2.1 功能需求

| # | 需求 |
|---|---|
| F1 | `GET /slist` 与 `GET /slist/:way` 均可访问,与 `/self` 路由形式一致 |
| F2 | `way` 既匹配自助模式坐标,也匹配代理模式坐标(自助+代理统一管理) |
| F3 | 公开接口,无需登录(与 `/self` 一致,管理后台另有 `/collection/geturls`) |
| F4 | 返回该 way 对应映射集的**完整 URL 数组**:每条含 id / parent / address / alive / retry |
| F5 | 负载字段名沿用现有 entity.Url 的 json tag(`address` 而非 path) |

### 2.2 非功能约束

| # | 约束 |
|---|---|
| C1 | 鉴权:公开,无 token/cookie 校验 |
| C2 | 一致性:与 `/self`、`/agent` 使用**同一份**内存缓存读取地址列表(不绕去 DB) |
| C3 | 命名:沿用现有包的命名习惯,本次不改 `M*` 互斥量前缀 |
| C4 | 失败响应风格:沿用现有 `{"code": N, "msg": "..."}`
| C5 | 不引入 schema 变更/迁移 |

### 2.3 显式不做(YAGNI,本次不做)

- 不在本接口实现心跳 / 自动启停地址(清单第 5 项,独立功能)。
- 不在本接口实现加盐模式(清单第 3 项)。
- 不在本接口实现手动启停地址(清单第 4 项;该功能会反映到本接口的 `alive` 字段,但入口在后端管理 UI)。
- 不在本接口增加过滤参数,如 `?onlyAlive=true` 之类。

## 3. 接口契约

### 3.1 请求

- 路由:`GET /slist`、`GET /slist/:way`(实际注册 `Any(...)`,以与 `/self` 同口径)。
- 参数:`way` 必填,通过 query `?way=abc` 或路径 `/slist/abc` 传入。

### 3.2 成功响应

```json
{
    "code": 0,
    "msg":  "success",
    "way":  "abc",
    "mode": "selfhelp",
    "point": "someCollection",
    "data": [
        { "id": 1, "parent": "someCollection", "address": "http://x", "alive": true, "retry": 0 }
    ]
}
```

字段说明:

| 字段 | 类型 | 含义 |
|---|---|---|
| `way` | string | 客户端传入的坐标 |
| `mode` | string | `selfhelp` 或 `proxy`,表示 way 取自哪类模式 |
| `point` | string | 实际指向的映射集名称(即 `collection.Name`) |
| `data` | array | 映射集中 URL 列表(空集合时为 `[]`,**不会**把 0 个地址当作 error 推 `code: 404`) |

### 3.3 失败响应

| 场景 | 响应 |
|---|---|
| `way` 缺失或空 | `{"code": 404}` |
| `way` 在自助/代理两个坐标缓存中都查无 | `{"code": 404}` |
| 命中的映射集里 URL 为空 | `{"code": 0, "data": [], ...}` |

## 4. 数据流与代码布局

### 4.1 读取链路

```
dispatchAny("/slist","/slist/:way")
  └─ slistHandler(c)                          // router/self_help_list_router.go
       └─ collection.GetSelfHelpList(way)      // collection/self_help_list.go
            ├─ MSelfHelpMode.Lock → 查 SelfHelpMode[way].
            │   命中 → MWorkCllection 取 WorkCllection[point] → 返回 (srcSelf, point, urls)
            ├─ MProxyMode.Lock → 查 ProxyMode[way].
            │   命中 → MWorkCllection 取 WorkCllection[point] → 返回 (srcProxy, point, urls)
            └─ 都没命中 → 返回 (srcNone, "", nil)
```

### 4.2 新增文件

- `collection/self_help_list.go`:`GetSelfHelpList(way string) (source listSource, point string, urls []entity.Url)`。
- `router/self_help_list_router.go`:`SelfHelpListRouter` 实现 `RouterTemplate`(Prepare/Init/Destroy)。

### 4.3 修改文件

- `main.go`:`RegisterRouter(...)` 入参增加 `&router.SelfHelpListRouter{}`。

### 4.4 不变的部分

- `entity/dao_entity.go` · `Url` 已包含 id/parent/address/alive/retry,无需改 schema。
- `dao/url_dao.go` · 读取走内存缓存,不再查 DB;无新增 DAO。
- 现有 `/self`、`/agent`、`/collection/geturls` 行为不受影响。

### 4.5 缓存一致性策略

复用与当前 `/self`、`/agent` 完全一致的缓存读取路径:`SelfHelpMode` / `ProxyMode` 两个坐标表 + `WorkCllection` 地址表,均在内存。

所有写路径(collection/self_help/proxy 的 add/update/delete)已各自维护缓存与 DB 的一致性。本次不新增写路径,故缓存一致性保持不变;`GetSelfHelpList` 仅新增一个读端点,与既有读端点等价。

互斥量约定(沿用现有代码):

- `MSelfHelpMode` 护 `SelfHelpMode`
- `MProxyMode` 护 `ProxyMode`
- `MWorkCllection` 护 `WorkCllection`

读端加锁顺序:先坐标表(`MSelfHelpMode` → `MProxyMode`),命中后再 `MWorkCllection`;避免同时持有多把。本实现严格按 unlock 再起下一把锁的方式,规避死锁。

## 5. 错误处理与边界情况

| 场景 | 处理 |
|---|---|
| `way` 空或缺失 | 立即 `{"code": 404}`,不查缓存 |
| way 同时命中自助与代理 | 优先自助(互通坐标本不应重复;优先级自洽即可) |
| 映射集存在于坐标但该集无 URL | 返回 `{"code": 0, "data": [], ...}` |
| 管理后台并发修改 URL | 读锁互斥,返回修改前或修改后快照之一——与现有 `/self` 行为等价,不引入更强一致性 |
| `/slist/:way` 路径形式与 `/slist?way=` 等价 | 通过同一个 util.GetWayParam 解析 |

## 6. 测试

### 6.1 单元测试(`collection`)

- `GetSelfHelpList`:
  - 已知自助 way → `(srcSelf, point, 与 WorkCllection[point] 一致的 urls)`
  - 已知代理 way → `(srcProxy, point, 与 WorkCllection[point] 一致的 urls)`
  - 未知 way → `srcNone`
  - 自助/代理同时命中同一 way → 返回 `srcSelf`

### 6.2 路由绑定测试

- `SelfHelpListRouter.InitRouter`:`/slist`、`/slist/:way` 在 `GET/POST/OPTIONS/...` 下均可命中,共持一个 handler。

### 6.3 HTTP 冒烟(真实路由加载)

- `GET /slist?way=<known-self-way>` → 200,`code=0`,`data` 非空数组。
- `GET /slist?way=<known-proxy-way>` → 200,`code=0`,`mode=proxy`。
- `GET /slist?way=<unknown>` → 200,`code=404`。
- `GET /slist` 缺 way → 200,`code=404`。
- 不携带任何 cookie/token,上述均应通过(公开接口)。

### 6.4 回归

- `/self`、`/agent`、`/collection/geturls` 行为与返回形态不因本次改动而变化。

## 7. 风险与备忘

- 当前 `url.Alive` 字段虽然有 schema,但尚无写入方(心跳/启停尚未实现)。故目前返回的全部为 `alive=true` 是**预期行为**,非 bug。
- 当后续「自动生效/失效地址(心跳检查)」(清单第 5 项)、「手动生效或失效地址」(清单第 4 项)实现时,将自然驱动 `alive` 字段,本接口无需改造即可反映真实健康。
- 本接口的真实价值在心跳/手启停实现后才会完全体现——本次先把查询口子铺好。

## 8. 实施步骤(概要,具体见后续 implementation plan)

1. 增补 `collection/self_help_list.go` 的 `GetSelfHelpList`。
2. 增补 `router/self_help_list_router.go` 的 `SelfHelpListRouter` 与 `/slist` `/slist/:way` 注册。
3. `main.go` 注册新路由。
4. 写单元 / 路由绑定 / HTTP 冒烟测试,跑全量回归。
5. 更新 README 开发进度清单,把 `/slist` 勾选为已完成。
