# 服务器携密钥自申请停用(可指定时间) 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 「 · 服务器携密钥自申请停用(可指定时间) 」

## 1. 背景

按 README 清单「服务器携密钥自申请停用(可指定时间)」#5。

现有能力:
- `url` 表有 `Alive bool` + `Retry int`(`entity/dao_entity.go::Url`)。
- `dao.UpdateUrlAlive(id, alive bool)` 写 DB + cache 双持(本次同期 lands)。
- `util.token_util::RandToken(n)` crypto/rand 62 字母表随机字符串。
- 心跳(#4) lands 同期: `beat` 60s 节拍,失败累 +3 → `UpdateAlive(id, false)`。
- 自申请停用(#5) 是独立的"维度"设施, 不应与心跳(#4)冲突。

本次目标: 服务器管理员在置阶段预持 URL 自密钥, 服务器自主调用入口请求下线一段时间(duration 秒), 到期自动恢复。

## 2. 功能 / 非功能需求

### 功能需求

| # | 需求 |
|---|---|
| F1 | 每条 URL 有唯一自申请密钥, 管理员通过后台接口获取/rotate |
| F2 | 服务器可 `POST /self-deactivate?id&key&duration` 请求下线一段时间 |
| F3 | duration 到期后自动恢复(`Alive=true`, `self_deactivate_until=NULL`, `Retry=0`) |
| F4 | 服务器可 `POST /self-activate?id&key` 提前恢复; 对 admin-手出禁用 url 返 403 |
| F5 | `duration<=0` 或 `>7200`(2h) 拒绝 |
| F6 | 管理员手动禁用的 url(self_deactivate_until=NULL) 自申请停用/激活 403 |
| F7 | 自申请恢复后心跳探活失败 → `alive=false`(带阻尼: 三次连续失败后放弃, 清 `self_deactivate_until=NULL`); url 锁定为 admin-manual 禁用, 需管理员介入 |
| F8 | 心跳(#4)不直接改 `self_deactivate_until` 或 `self_deactivate_attempts`; 心跳仅写 `alive + retry`; 负责恢复维度的只有 scheduler(#5) |

### 非功能约束

| # | 约束 |
|---|---|
| C1 | schema 扩 3 列: `self_deactivate_key`, `self_deactivate_until` `*time.Time` (指针 nullable), `self_deactivate_attempts int` |
| C2 | xorm.Sync2 在启动时 alter table add 三列(new DB ok 老 DB 自动补) |
| C3 | 时钟偏移容忍(绝对 timestamp, 重启后 scheduler 续拾) |
| C4 | 自申请 API 鉴权: key match(公开, key 即 secret); 后台管理 API: LoginHandler |

### 显式 YAGNI(本次不做)

- 不做告警(#6 单列)。
- 不做心跳/自申请维度以外的启停维度。
- 不做 master-secret derivation。path of实施把 key 存库。
- 不把 self-deactivate duration 扩到 maxDuration 2h 以上。
- 不为每个 URL 初始即生成 key(按需: 仅 GET /self-deactivate/key?id 时首次 seed)。

## 3. API 契约

**4 个端点**(后 2 个 Admin 端点 LoginHandler, 前 2 个公开端点 key-gate):

| 端点 | 鉴权 | Body/Query | Success(200) | Error(200 + code:1) |
|---|---|---|---|---|
| `POST /self-deactivate` | key | `{id:int, key:string, durationSec:int}` | `{code:0, msg:"ok", "deactivateUntil":"RFC3339"}` | `code:1`invalid param / `/key mismatch`/`disabled by admin`/`url not found` |
| `POST /self-activate` | key | `{id:int, key:string}` | `{code:0, msg:"ok"}` | `/403 url administratively disabled`/`key mismatch` |
| `GET /self-deactivate/key` | Login | `?id=int` | `{code:0, data:{key, url{id,path,alive,deactivateUntil,attempts}}}` | `/404 url not found`/`1 Admin miss` |
| `POST /self-deactivate/key/rotate` | Login | `?id=int` | `{code:0, data:{key}}`(new key) | `//not found`/`internal` |

## 4. 数据流 + 代码位置

### 扩表(url)

`url` 表加 3 列:

| 列名 | 类型 | 默认 | 说明 |
|---|---|---|---|
| `self_deactivate_key` | varchar(200) NOT NULL DEFAULT '' | 空字符串 | URL 自申请密钥, 管理员 GET 时若空则自动 seed |
| `self_deactivate_until` | timestamp NULL | NULL | 到期时间(NULL = 无在用自申请) |
| `self_deactivate_attempts` | int NOT NULL DEFAULT 0 | 0 | 自恢复后连续心跳失败次数 |

### 改动文件

| 文件 | 类型 | 职责 |
|---|---|---|
| `entity/dao_entity.go::Url` | 改 | 加 3 字段 |
| `dao/url_dao.go` | 改 | 加 `GenerateAndSaveUrlKey(id) (string, error)` + `SetUrlSelfDeactivate(id, until, attempts)` + `ClearUrlSelfDeactivate(id)` |
| `router/setting_router.go` | 改 | 加 4 handler (public: self-deactivate / self-activate; admin: get key / rotate key) |
| `scheduler/reactivate_scheduler.go` | 新 | register `beat()` 每 60s 扫一遍 `alive=false AND self_deactivate_until IS NOT NULL AND self_deactivate_until <= now` → 恢复逻辑(url + attempts 阻尼) |
| `scheduler/scheduler_register.go` | 改 | 注册 registerReactivateScheduler |

### 自申请写入语义

`POST /self-deactivate`:
```go
// 1. duration 校验: else 0
// 2. id 取 url(走 cache 或 db)
// 3. key 比较(url.SelfDeactivateKey == body.Key)
// 4. 如果 url.Alive=false 且 url.SelfDeactivateUntil==nil → admin 手出禁用 → 403 return
// 5. url.SelfDeactivateUntil = now+duration
// 6. dao.UpdateUrlAlive(id,false) + dao.SetUrlSelfDeactivate(id, until, 0)
// 7. return 成功 deactivateUntil = until.Format(time.RFC3339)
```

`POST /self-activate`:
```go
// 1. id 取 url
// 2. key 比较
// 3. 如果 url.SelfDeactivateUntil==nil → 没在 self 停用中 → 400("url not self-deactivated")
// 4. 如果 url.Alive=false 且 url.SelfDeactivateUntil==nil(但上一条 pass 已经否掉, 这个不会 hit)
//    额外 validation: 如果 url 已被 admin 手出禁用, 本 API 不放行 → 403("url is administratively disabled; contact admin")
// 5. dao.UpdateUrlAlive(id,true) + dao.ClearUrlSelfDeactivate(id)
// 6. return ok
```

### reactivate scheduler(核心)

**reactivate_scheduler.go::beat()** 每次 beat:
```go
// 1. 扫 url 表: SELECT id FROM url WHERE alive=false AND self_deactivate_until IS NOT NULL AND self_deactivate_until <= now()
// 2. 逐条 url:
//    a. attempts = url.SelfDeactivateAttempts
//    b. if attempts >= 3 { ClearUrlSelfDeactivate(id); log("[reactivate] url id=%d attempts=%d -> give up", id, attempts); continue }
//    c. UpdateUrlAlive(id, true)    // Alive=true, Retry=0
//    d. SetUrlDeactivateAttempts(id, attempts+1) // 心跳探失败后再次 down, 下轮 beat 看是否再恢复
// 3. 成功恢复 url 的 SelfDeactivateUntil 保留到 next beat(阻尼窗口)
//    实际上下轮 beat 如果 self_deactivate_until > now, scanner 就不会 pick 它;
//    一旦 self_deactivate_until<=now 并且现在 alive=true(由心跳再次 down 前), scanner 不会 pick(因为 alive=true)
//    → 而心跳 down 后 scanner 下一拍又 pick(if still until<=now). 这是正则阻尼。
```

**阻尼循环**:
- T0: self-deactivate until=T0+1h, attempts=0
- T0+1h: beat → release attempts=1, alive=true
- 心跳下一拍探失败 → alive=false
- T0+1h+1min(下一 beat) → scanner → attempts 1 < 3 → 再 release attempts=2
- 心跳再 down
- T0+1h+2min → attempts 3 → 放弃, clear self_deactivate_until, log"give up, needs admin"
- url 现在是 admin-effective 禁用(alive=false, until=null, attempts=0)

收敛 O.K.

## 5. 错误处理 + 边界

| 场景 | 处理 |
|---|---|
| durationSec<=0 | `{"code":1,"msg":"duration must be >0"}` |
| durationSec>maxDuration(=7200) | `{"code":1,"msg":"duration too large(max 7200s)"}` |
| key miss(url 不存在) | 400-class msg "invalid key or url"(平等文案防 id 枚举) |
| url.Alive=false 且 until=NULL(self-activate 进来) | 403 "url is administratively disabled; contact admin" |
| url.Alive=true, until=NULL 时 self-activate | 400 "url not self-deactivated" |
| 重复 self-deactivate | 续新 until 或 同一 until(等义) |
| 时钟偏移/重启 | scheduler 续拾所有 due url |
| URL 密文 key 被管理员新 rotate | 旧 key 立刻失效; 持有旧 key 的服务器进一步请求 403 |
| admin-enable 与自申请窗口冲突 | 若 url 正处于自申请窗口内, 管理员调用 `/url/enable`(或任何 admin-enable 路径), 必须 clear 自申请三字段(`self_deactivate_until=NULL, attempts=0`)。这保证「管理员说了算」: admin-enable 一出, 自申请窗口立刻销毁。任何 Application 层调用 `dao.UpdateUrlAlive(id, true)` 成功后的 downstream handler 都必须 `ClearUrlSelfDeactivate(id)`。此 invariant 严禁下放到 `UpdateUrlAlive` dao 函数。|

## 6. 测试

1. `dao.url_self_deactivate_test.go`: GenerateAndSaveUrlKey 返 non-empty + 写库 + cache 同步。Set/clear 三字段写 + 读。
2. `router.setting_router_test.go`(inline httptest): 模拟 4 handler 的 success + error path(禁 key 403 / 禁用 url 403 / 没 url-没 url 400 / 平等错误文案)。
3. `scheduler.reactivate_scheduler_test.go`: mock url seed(until 已过), 跑一轮 beat 断言 alive=true + attempts+1; 连续 N 次心跳 down + beat 断言 N 次后 attempts=3, until=NULL。

## 7. 风险与备忘

- 阻尼列(方案 A)不加 disable_source enum — 使用 `self_deactivate_until NULL` 隐式编码禁用来源。
- 心跳(#4)与自申请(#5) 维度重合在 `url.Alive` 一个字段上; 最终收敛由 beat/reactivate scheduler 协作保证。
- admin-enable 与自申请的关系 — Application 层 invariant: 任何 admin-enable 操作都需 clear 自申请三字段。详见 error 表项「admin-enable 与自申请窗口冲突」。
- self-deactivate key 生成: 首 GET `/self-deactivate/key?id` url 时若 `SelfDeactivateKey==''` 则 seed + 持久化, 随后 rotate。
- xorm 1.4 `Sync2(&url{})` 对老 DB 自动 ALTER TABLE ADD COLUMN; 新增三列带 default, 不需要 migration script。
