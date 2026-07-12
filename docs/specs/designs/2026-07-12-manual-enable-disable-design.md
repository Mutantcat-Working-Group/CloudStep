# 手动生效/失效地址 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 开发进度「手动生效或失效地址」

## 1. 背景与目标

「云阶」项目 README 「开发进度」清单中「手动生效或失效地址」一项标记为未完成。

现有能力:
- `url` 表已有字段 `Alive bool` 和 `Retry int`(定义见 `entity/dao_entity.go::Url`, schema 已存在于 `cloud_step.db`)。
- 管理后台 `/collection/geturls` 只读返回地址列表(含 `alive` / `retry`), 但**没有任何写入接口**。
- 读端点 `GetPath`(`self_help_mode.go`) 和 `GetProxyPath`(`proxy_mode.go`) **在随机/轮询时不跳过 `Alive=false` 的地址** — 地址即便停用也可能被选到(一个已有 bug)。

本次目标: 让管理员可在后台手动启用/禁用某条地址; 顺带修复 `GetPath` / `GetProxyPath` 跳过死地址; 心跳检查(#4)未来复用同一条事务路径。

## 2. 功能 / 非功能需求

### 功能需求

| # | 需求 |
|---|---|
| F1 | 新增 `POST /url/enable` 和 `POST /url/disable`, 均为登录鉴权 |
| F2 | 请求体 `{ "id": <int> }`, Id>0 才处理 |
| F3 | `enable` 把 `url.Alive=true` **并把 `url.Retry` 复位为 0** |
| F4 | `disable` 把 `url.Alive=false`, **不动** `url.Retry` |
| F5 | 保持 DB 与内存 cache `WorkCllection` 一致 |
| F6 | 后端 checkbox/开关, 点一次调一次; 不改现有 `/url/add` `/url/delete` `/url/update` UI |
| F7 | `GetPath` / `GetProxyPath` 跳过 `Alive=false`; 全死则返回 ""); `collection.GetSelfHelpList` 列表**不动** |
| F8 | 心跳(#4)未来调用, 与本期共用 `UpdateUrlAlive(id, alive)` 事务 |

### 非功能约束

| # | 约束 |
|---|---|
| C1 | 单行原子:事务里一次 `UPDATE url SET alive=?, retry=? WHERE id=?` |
| C2 | 与现有名称 `/url/*` 一致(单操作单端点) |
| C3 | 启用/禁用都走同一 DAO 函数 `UpdateUrlAlive(id int, alive bool)`, 不拆两个 |
| C4 | 启用时 Reset Retry 是 idempotent 语义 — 若已经是启用态且 Retry=0 仍返回 `code:0` |
| C5 | 心跳(#4) 与 管理员 共用写入, 不做差异化存储 |
| C6 | 升级 /agent /self 过滤: `GetPath` / `GetProxyPath` 内新增 `filterAlive` helper |

### 显式 YAGNI (本次不做)

- 不做「自动 / 心跳 / 管理员」启停来源区分(见 C5)。
- 不做 后台 UI 重构 — 只在现有映射集详情表格里加一个启用/禁用切换。
- 不做 启停历史审计(谁、什么时候、为什么停用)。
- 不做 心跳(#4) 本身, 只把共用路径铺好。
- 不做 `/slist` 列表过滤 — 列表是给人看真实健康态(Debug view)。

## 3. API 契约

### 3.1 `/url/enable`

```
POST /url/enable
Authorization: 登录 cookie / 通过 LoginHandler()
Content-Type: application/json

Body:
{ "id": <int> }
```

| 响应 | 条件 |
|---|---|
| `{"code":0,"msg":"success"}` | id>0 且 DB 中存在该行 |
| `{"code":1,"msg":"error"}` | id 缺失/≤0 或不存在 |

副作用: `url.Alive=true, url.Retry=0` 写 DB + cache 同步。

### 3.2 `/url/disable`

请求体和响应形状同上。副作用: `url.Alive=false` 写 DB + cache, `url.Retry` 不动。

### 3.3 idempotent

再用同一个 id 调 `enable` 仍返回 `code:0` (Retry 再次 = 0, 语义无损)。调 `disable` 同理。

## 4. 代码位置 + 数据流

### 修改 / 新增文件

| 文件 | 类型 | 职责 |
|---|---|---|
| `router/setting_router.go` | 修改 | 在 `/url/delete` 后追加 `POST /url/enable` `POST /url/disable` 路由与 handler |
| `dao/url_dao.go` | 修改 | 新增 `UpdateUrlAlive(id int, alive bool) bool` 事务 + cache 同步 |
| `collection/self_help_mode.go` | 修改 | `GetPath` 内调用 `filterAlive` |
| `collection/proxy_mode.go` | 修改 | `GetProxyPath` 内调用 `filterAlive` |
| `entity/dao_entity.go` | **不动** | `Url{Alive,Retry}` 字段已存在, 无需 schema 改动 |

### 数据流

```
POST /url/enable {id: N}
  └─ enableHandler (LoginHandler gate)
       └─ dao.UpdateUrlAlive(N, true)
            ├─ session.Begin()
            ├─ session.Cols("alive","retry").ID(N).Update(&Url{Alive:true, Retry:0})
            ├─ session.Commit()
            └─ cache 同步:
                 MWorkCllection.Lock()
                 for coll, urls := range WorkCllection:
                     for i in urls: if urls[i].Id == N { Alive=true; Retry=0 }
                 MWorkCllection.Unlock()

POST /url/disable {id: N}
  └─ disableHandler (LoginHandler gate)
       └─ dao.UpdateUrlAlive(N, false)
            ├─ session.Begin()
            ├─ session.Cols("alive").ID(N).Update(&Url{Alive:false})
            ├─ session.Commit()
            └─ cache 同步 (同上, 只改 Alive, 不动 Retry)
```

`GetPath` 修订:

```
selfHelpMode.GetPath(way):
  ... 按现有方式拿到 WorkCllection[point] (加锁内) ...
  WorkCllection[point] 引用释放后:
    可活 := filterAlive(WorkCllection[point])
    若 len(可活) == 0: return ""
    用 可活 选下一个(random/polling 索引在可活内 roll)
```

`filterAlive` 写在各自文件里保持文件边界; 两份 helper 逻辑相同但接受独立演化(心跳未来可能引入"探活重试计数"让 helper 分化)。

### xorm bool zero-value 实现备忘(关键)

`session.Update(&Url{Alive: alive})` 的 xorm 默认行为是只写 non-zero 字段。`false` 是 bool zero, 会被跳过 → 禁用时将写不进去。正确写法是强制列:

- **启用(`alive=true`)**: `session.Cols("alive","retry").ID(id).Update(&entity.Url{Alive:true, Retry:0})`
  → DB 修改 alive=true, retry=0。
- **禁用(`alive=false`)**: `session.Cols("alive").ID(id).Update(&entity.Url{Alive:false})`
  → 只强制 `alive` 一列, xorm 仅改它; `retry` 不出现也不被触及, 自然满足"禁用时不改 Retry"。

即 **两场景强制列不同**: enable 强制两列, disable 强制一列。这是同一函数 `UpdateUrlAlive(id, alive bool)` 里用 `if alive { …Cols("alive","retry")… } else { …Cols("alive")… }` 分支的原因。

各调用方(handler)不关心列细节, 只调 `dao.UpdateUrlAlive(id, alive)`:

## 5. 错误处理 + 边界

| 场景 | 处理 |
|---|---|
| 缺 id 或 id ≤ 0 | 立即 `{"code":1}` |
| id 在 DB 中不存在 | `session.ID(id).Update(...)` 影响行 = 0 → 返回 `{"code":1}`(与现有 `/url/update` 同口径) |
| 已经是目标状态 | 仍走事务(reset retry) → `code:0`, idempotent |
| 映射集全 `Alive=false` | `GetPath` / `GetProxyPath` 返回 "" → handler 现兜底 `{"code":404}`(与 way 不存在同形) |
| 并发 admin 启停 + #4 心跳探活 | 共用 `UpdateUrlAlive`, xorm 单行原子 + `MWorkCllection` 写锁; 不区分来源 |
| cache 与 DB 不一致(重启) | 现有代码 `initModes()` 启动时从 DB 全量重载 cache, 重启后恢复一致 |
| xorm `Cols` 写 | 如上: true 启用 → `Cols("alive","retry")` 写 alive+retry; false 禁用 → `Cols("alive")` 只改 alive, retry 不动 |

## 6. 测试

### DAO 层

- `dao.UpdateUrlAlive(id, true)`:
  - 对存在的 id: `alive=true, retry=0` 写 DB + cache `WorkCllection` 中同 Id 的 url 检查 alive/retry
  - 对不存在的 id: 返回 false
- `dao.UpdateUrlAlive(id, false)`:
  - 对存在的 id: `alive=false` 写 DB, retry 不改; cache 同步
  - 事务错误场景: 由 Commit 返回 false(单层测试不必 mock)

### handler 层

- `POST /url/enable` + 未登录 → 由 `LoginHandler()` 拦截, `code:1` 未登录
- `POST /url/enable` + id=0 → `code:1`
- `POST /url/enable` + id=有效 → `code:0`
- `/url/_disable_ 同理

### 读端点修订

- `collection.GetPath` 在至少一个 Alive=true 地址时仍选到一个 Alive=true 的
- `collection.GetPath` 全 Alive=false 时返回 ""
- `collection.GetProxyPath` 同上
- `collection.GetSelfHelpList` 仍包含 `Alive=false` 的条目(列表是 debug view)

### 回归

- `/url/add` `/url/update` `/url/delete` 行为未变
- `/self`、`/agent`、`/slist` 端点仅加了过滤, 既有契约未变

## 7. 风险与备忘

- **启用复位 Retry=0 语义**: 未来心跳(#4)共用此路径时, 心跳判活成功 → `enable` 会复 retry=0, 这是我们所期望的。心跳判死 → `disable` 不改 retry, 所以 retry 数累计用于告警(#6)。
- **`filterAlive` 双文件独立实现**: 未来心跳/自申请停用可能分化 helper, 双文件边界给演化留缝。
- **前端改动**: 在管理后台(`web/cloud-step-web-1g/...`)映射集详情表格的每行 URL 加一个启用/禁用按钮。后端接口先独立落地; 前端是否在本轮同步升级未在 spec 内规定, 需单独前端工作。本轮 spec 仅管 Go 后端。
