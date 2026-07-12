# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.20260711] - 2026-07-11

### Dependency upgrades (安全/漏洞防线)

## [1.0.20260712] - 2026-07-12

> 本轮在未发布的 `feat/empty-path-default-mode` 分支上推进了 3 项功能与 1 项安全落地。

### Added(手动生效/失效地址)

- 新增 `POST /url/enable` 与 `POST /url/disable`(均 LoginHandler 鉴权), 后台管理员可手动启停单条 URL。`enable` 同时复位 `url.retry = 0`;`disable` 不改 `retry`(累积计数用于心跳告警观察)。
- 新增 `dao.UrlDao::UpdateUrlAlive(id int, alive bool) bool`: 事务写 `url.alive` + `url.retry`(enable 时清零), 同步内存 `WorkCllection` 缓存; 供手动启停与心跳共用同一写入链路, 彻底从根源解决 DB / cache 不一致。
- 扩展 `GET /collection/geturls` 返回的 `url` JSON 透出 `id / parent / address / alive / retry` 字段, 后台可观察 URL 健康状态与失败计数(本次 `/slist` 已前置透出, 本次再为 `/collection/geturls` 打通)。
- 新测试: `dao/url_state_test.go` (5 个 case, 含 self/proxy/unknown/emptyCollection/collision 路径)。

### Added(自动启停心跳)

- 新增 `scheduler/heartbeat_scheduler.go`: 每 60s 对所有 `Alive=true` 的 URL 并发 TCP 500ms 探活(`util.GetTCPSpeed`); 失败 → `url.retry + 1`; 达 N=3 → 调 `dao.UpdateUrlAlive(id, false)` 下线 + stderr 事件日志 `[heartbeat] url id=... set ALIVE=false after N failures`; 成功 → `url.retry` 清零。管理员手动禁用的 URL(`Alive=false` 无论来源)一拍全跳过, 心跳**只摘不复活**, 与手动启停互不踩。
- 新增 `dao.UrlDao::UpdateUrlRetry(id int, targetRetry int) bool`: 写入 `url.retry` 字段到目标值(幂等, 已等值省 IO), 同步 `WorkCllection` 缓存; 心跳失败/成功两条路径复用同一函数(targetRetry 由调用方针算)。心跳计数器跨重启持久(`url.Retry` 字段落盘), 比纯内存重启清零更贴「故障持续」现场语义。
- 新注册到 `scheduler/scheduler_register.go`。符合 antlabs timer `ScheduleFunc(dur, fn)` 节奏。
- 新测试: `scheduler/heartbeat_scheduler_test.go` (2 个 case: 探活成功清零 + 连续 failThreshold 次探失败下线), 本地 httptest server 隔离。

### Added(空路径默认模式)

- `GET /sysconfig/get` 响应新增两个字段 `selfDefaultCollectionId` / `agentDefaultCollectionId`(`int`, 0 表示未配置)。
- `POST /sysconfig/update` 请求体扩同两字段: id>0 时校验 `collection` 表存在, 不存在返 `code:1, desc:"... not found"`; id=0 视为"清除默认"。
- `GetPath` / `GetProxyPath` 顶部新增兜底逻辑: `way==""` → 通过新增 `util.ResolveWayCollection(way, defaultId)` helper 落到对应默认映射集; `way` 有值完全不读默认配置, 现有契约零破。
- `self_help_mode.go::filterAlive` / `proxy_mode.go::filterProxyAlive`(Go 拒绝同 package 同名 helper,故两名)跳过 `Alive=false` 条目; 全死返 `""` → handler 返 `code:404`。`/slist` 列表端点 unchanged(列表为 debug view)。
- `entity.SystemConfig` 加两个字段; `util/system_config.go` 重构为持有 `entity.SystemConfig` 镜像加 RLock 只读访问器 `GetSysConfigMirror()`(热路径绕过 DB); `AllowIntranet()` 向后兼容 shim 保 `./ip.go` 单调用点 unchanged。
- 新 helper `util.ResolveWayCollection`: 通过 resolver 注入模式(不在 `util → dao` 形成 circle; `dao.SetDefaultCollectionResolver(GetCollectionNameById)` 在 `InitSystemConfig` 冷启动注入)查 collection 名。新测试 `util/resolve_default_test.go` (mock resolver, 4 case)。
- `lifecycle/RegisterRouter` 入参追加 `&router.SelfHelpListRouter{}`。
- 新测试: `collection/slist_test.go` (`GetSelfHelpList` 5 case) / `collection/url_state_read_test.go` (`GetPath` / `GetProxyPath` 4 case) / `util/system_config_test.go` (1 case) / `router/slist_router_test.go` (5 case Any-verb + query/path 全绑定)。

### Added(服务器携密钥自申请停用——设计已定, 本轮实施待完成)

- 设计见 `docs/specs/designs/2026-07-12-self-deactivate-design.md`。spec 定稿:URL 粒度 per-URL 密钥(存库, `crypto/rand`);`url` 表扩 3 列 `self_deactivate_key` / `self_deactivate_until`(`timestamp NULL`) / `self_deactivate_attempts`;新增 4 handler `POST /self-deactivate` `POST /self-activate` (key-gated) + `GET /self-deactivate/key` `POST /self-deactivate/key/rotate` (Login admin);scheduler `registerReactivate` 每 60s 到期自动恢复 + 阻尼 3 次连续心跳失败后放弃并清 `until`(locking URL 为 admin-effect, 需管理员介入)。
- 本轮仅完成 design spec (commit `c3ce3ff`); 实现 + 测试 + README 勾选待 next session 实施。

### Fixed

- `SelfHelpMode.GetPath` / `ProxyMode.GetProxyPath` **此前不跳过 `Alive=false` 条目, 死地址会被轮询/随机选到**, 现已修, 直面("手动/心跳下线后还能选到死地址")真实 bug。
- `util.GetWayParam` 内 `c.BindJSON` 在 path-form 空 body 请求时会把 400 写进 response writer 再返回 error, 导致 `/slist/abc` 类 path-form 全部返回 400。处理: handler 实现本地 `resolveWay(Query → PostForm → Param("name")`, JSON 仅当 `Content-Type: application/json` 尝试), 不动公共 util, 不破坏 `/self` `/agent` 行为。

### Security

- `entity system_config` 新字段 id 写入前背调 `dao.GetCollectionNameById`, 不存在返错; 防止管理员误配导致默认集为 dangling id 后默默 404。
- 手动启停端点 `/url/enable` / `/url/disable` 走 `LoginHandler` 鉴权, 未登录一律 `code:1,msg:"未登录"`。
- 心跳探活节拍 + 阈值 + 并发: 百级 URL 单次 beat 嵌 60s 节拍安全, 无 goroutine leak via `sync.WaitGroup`。

### Changed

- **`/slist` 是公开端点(无 LoginHandler), 返回完整 URL 数组供frontend 做自助负载均衡决策**, 契约对齐 README 三的「自助模式」§3 节意图。
- `util/ping_util.go::GetTCPSpeed` 本轮被复用两次(手动 Ping 端点 + 心跳), util 生态一致性提升。

---

## [1.0.20260711] - 2026-07-11

### Dependency upgrades (安全/漏洞防线)

- `github.com/gin-gonic/gin` v1.9.1 → **v1.12.0**(次要版本升级,基础 API 保持向后兼容)。
- `github.com/gin-contrib/cors` v1.7.0 → **v1.7.7**。
- `github.com/mattn/go-sqlite3` v1.14.22 → **v1.14.47**(捆绑的 SQLite C 库更新,覆盖多个上游 CVE)。
- `github.com/antlabs/timer` v0.1.3 → **v0.1.4**。
- `xorm.io/xorm` v1.3.8 → **v1.4.1**(保持向后兼容,未改代码接口)。
- 同步清理 `go.mod` / `go.sum`,拉齐 transitive 依赖(`golang.org/x/*`、`google.golang.org/protobuf`、`github.com/goccy/go-json`、`github.com/pelletier/go-toml/v2`、`github.com/ugorji/go/codec`、`github.com/go-playground/validator/v10`、`github.com/gabriel-vasile/mimetype` 等)。

### Security

- `util/ping_util.go` `urlToIPP` 修复越界崩溃(DoS):对长度不足 7/8 字节的输入做边界检查,并剥离 `user:info@` 用户信息段,避免服务端直接 panic。(#1)
- `lifecycle/gin_service.go` 修复 `RegisterRouter` 中 `InitRouter` 错误被赋值吞掉的问题,改为累计返回,确保后续路由初始化失败可被上层感知。
- `util/proxy_util.go` 修复 `RootProxy` 查询参数注入:使用 `url.Values` 收集并对参数进行 URL 编码,防止用户输入中的特殊字符被注入为额外参数;`Proxy` 增加目标地址解析失败的校验。
- `util/token_util.go` 将登录 token 生成算法从 `math/rand + 时间种子` 替换为 `crypto/rand`,长度与字符集保持不变,消除 token 可预测风险。
- 新增可开关的 SSRF 反代隔离机制:
  - 新增 `util/ip.go`,基于标准库 `net/netip` 对目标做 IP 分类(私有 / 回环 / 链路本地与云 metadata / 未指定)。
  - 新增 `entity.SystemConfig`、`dao.SystemConfigDao`,新增数据库表 `system_config`,单列记录运行时开关。
  - 新增 `util/system_config.go` 内存镜像与读写锁,启动时从 DB 加载。
  - 新增后台接口 `GET /sysconfig/get` 与 `POST /sysconfig/update`(需登录),可在「系统管理」中切换。
  - `Proxy`/`RootProxy` 在开关关闭时拒绝代理到内网 / 私有 / 回环 / 链路本地 / metadata 地址;默认 `allowIntranetProxy=true`,保留内网 / 智能家居场景。

### Added

- CHANGELOG.md 项目变更日志。
- VERSION 项目版本号文件。

## [1.0.20240329] - 2024-03-29

### Added

- 实现了基本的功能。

---

## 后续修复(待实施,本次及后续迭代处理)

> 以下内容已经过安全审计识别,尚未实施,后续按用户指令迭代修复:

- **密码存储升级为哈希**:当前用户密码以明文形式存储在 SQLite,默认账号 `admin96/admin96`。后续修复:使用 bcrypt/argon2 哈希存储,并在登录页添加强制要求首次登录修改密码。
- **多用户 / 多 token 体系**:当前登录态为包级全局 `token` 单变量,所有登录用户共享;修改密码与重新登录会覆盖先前的 token。后续修复:引入 session 表、支持多设备独立 token 与过期失效。
- **后台 CORS 配置收紧**:当前 `AllowAllOrigins = true`,后续修复为默认仅允许自身 origin 或后台可配置白名单。
- **代理错误响应去敏感化**:当前代理失败时直接将 `err.Error()` 回写响应,可能暴露内部网络拓扑。后续修复:对外返回统一错误文案,详情仅记录到服务端日志。
- **登录限速持久化与账户锁定**:当前 `util/login_times_util.go` 的限频计数仅存在于内存且按全局阈值、每 3 分钟清空,重启即重置。后续修复:限频计数持久化到数据库、支持失败次数阈值与账户临时锁定。
- **前端切密码页新旧密码校验与强度校验**:当前 `change` 接口接受任意密码,后续在前端补充二次确认与最低强度校验(长度、字符种类)。
