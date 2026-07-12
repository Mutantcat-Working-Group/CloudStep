# SSL 证书支持(管理员配置 cert/key 路径) 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 「SSL 证书支持」#7

## 1. 背景

按 README 清单「SSL 证书支持」#7。CloudStep 当前 `lifecycle/gin_service.go::StartGin` 仅通过 `ginServer.Run(":"+port)` 起 HTTP 单栈,Gin 原生同时支持 `RunTLS(addr, certFile, keyFile)`,无需引入额外依赖即可在管理员配置 cert/key 文件路径后开启 HTTPS 入口。

本次目标:管理员在后台配置 SSL 总开关 + cert 文件路径 + key 文件路径 + HTTPS 端口;下次冷启动 `StartGin` 读取配置并按「HTTPS-only 启用 / HTTP-only 未启用」的最小可行语义起服务。证书文件由运维放置在宿主机任意路径(容器内映射),程序只读 `os.Stat` 校验存在 + Go `crypto/tls` 加载,不内嵌签发能力。

## 2. 功能 / 非功能需求

### 功能需求

| # | 需求 |
|---|---|
| F1 | `system_config` 表扩 4 列: `ssl_enabled`(bool 总开关,默认 0)、`ssl_cert_path`(varchar 500,默认 '')、`ssl_key_path`(varchar 500,默认 '')、`ssl_port`(int 默认 9443,与 HTTP 9091 避让) |
| F2 | `entity.SystemConfig` 同步扩 4 字段(json camelCase 标签供 admin API) |
| F3 | `dao.UpdateSystemConfig(...)` 显式把这 4 列加入 `Cols(...)` 写入列表 |
| F4 | `GET /sslcerts`(LoginHandler)返回 `{enabled, certPath, keyPath, port}`(路径原样透出——路径本身非 secret,仅 admin 可见) |
| F5 | `POST /sslcerts/update`(LoginHandler)提交 `{enabled, certPath, keyPath, port}`;`enabled=true` 时两个路径必须均非空 **且** 文件在磁盘存在(`os.Stat`),否则返 `code:1, msg:"cert or key file not found"` |
| F6 | `StartGin` 改造:读 `util.GetSysConfigMirror()`,当 `SslEnabled && SslCertPath!="" && SslKeyPath!=""` 时启动 HTTPS goroutine(地址 `:<SslPort>` 调 `RunTLS`),HTTP 仍走 `Run(":"+port)`;否则行为不变(HTTP-only on `port`) |
| F7 | 默认 HTTPS 端口 9443,避免与 HTTP 9091 冲突 |
| F8 | 测试:`router/ssl_admin_test.go` 通过 httptest 覆盖 happy(自签 cert 临时文件)、missing-file(code:1)、disabled-allowed(空路径 OK)、unauthenticated 四个 case |

### 非功能约束

| # | 约束 |
|---|---|
| NF1 | **零依赖**:仅用 Go 标准库(`crypto/ecdsa` + `crypto/x509` + `pem.EncodeToMemory` 测试内自签),不引入第三方 |
| NF2 | **兼容性**:默认 `ssl_enabled=0` 时行为与改造前完全一致;既有部署无感知 |
| NF3 | **最小可行**:启用后 HTTPS-only,不再并行 HTTP(避免引入"浏览器跳转"等过度设计);前端要并行可后续加 |
| NF4 | **文件存在校验**:在写 DB 前 `os.Stat`,提前给出明确错误,而不是等 `RunTLS` 启动期 panic/log |
| NF5 | **goroutine 卫生**:HTTPS `RunTLS` 必须跑在 goroutine(否则阻塞 HTTP 启动),错误只 `log.Printf`,不 throw |
| NF6 | **测试隔离**:不用真 9443 端口/真 listen,仅验证 handler + DB 侧(`httptest.NewRecorder` 直接驱动 handler) |

## 3. 数据模型变更

`entity.SystemConfig`(单行 id=1)追加:

```go
SslEnabled  bool   `xorm:"notnull default(0)" json:"sslEnabled"`
SslCertPath string `xorm:"varchar(500) notnull default('')" json:"sslCertPath"`
SslKeyPath  string `xorm:"varchar(500) notnull default('')" json:"sslKeyPath"`
SslPort     int    `xorm:"notnull default(9443)" json:"sslPort"`
```

xorm 标签中的列名会自动派生为 `ssl_enabled` / `ssl_cert_path` / `ssl_key_path` / `ssl_port`(蛇形),与既有列命名风格一致。`varchar(500)` 给容器内长路径留余量;`notnull default` 保证老行零改动(ALTER TABLE ADD COLUMN 取 defaults)。

## 4. 接口设计

### `GET /sslcerts`
- 鉴权: `LoginHandler()`
- 响应:
```json
{ "code":0, "msg":"success", "data": { "enabled":false, "certPath":"", "keyPath":"", "port":9443 } }
```

### `POST /sslcerts/update`
- 鉴权: `LoginHandler()`
- 请求体:
```json
{ "enabled":true, "certPath":"/etc/ssl/c.pem", "keyPath":"/etc/ssl/k.pem", "port":9443 }
```
- 校验: 若 `enabled==true` 且 (`certPath==""` 或 `keyPath==""` 或 `os.Stat` 任一失败) → `code:1, msg:"cert or key file not found"`,**不落库**。
- 成功后落库(`dao.UpdateSystemConfig`)并同步 `util` 镜像,返 `code:0`。

## 5. 代码落地

| 文件 | 动作 |
|---|---|
| `entity/dao_entity.go` | `SystemConfig` 加 4 字段 |
| `lifecycle/gin_service.go` | 改 `StartGin`:加 `fmt`/`com.mutantcat.cloud_step/util` import;HTTPS 走 `RunTLS` 主栈,否则走 `Run` |
| `dao/system_config_dao.go` | `UpdateSystemConfig` 显式 `Cols(...)` 加 4 列(保持既有「三字段 + 事务 + 同步镜像」语义) |
| `router/ssl_admin_router.go` | 新建,RouterTemplate;2 handler |
| `router/ssl_admin_test.go` | 新建,httptest 4 case |
| `main.go` | `RegisterRouter(...)` 加 `&router.SslAdminRouter{}` |
| `docs/specs/designs/2026-07-12-ssl-cert-support-design.md` | 本文件 |
| `README.md` ~L91 | 勾选 `SSL证书支持` → `[X]` |
| `CHANGELOG.md` | 加 `### Added(SSL 证书支持)` 段 |

### `StartGin` 落地(严格一栈,避免 main 退出 / 端口漂移)

设计原则:「HTTPS-only 启用 / HTTP-only 未启用」。启用 SSL 时主栈走 `RunTLS` 不再并行 HTTP(旧客户端需切到 HTTPS `SslPort`,运维在 CHANGELOG 提示);未启用时行为与改造前完全一致。解析 `RunTLS` 必须阻塞主栈,否则 main 退出,因此不再使用 goroutine 旁路。

```go
func StartGin(ginServer *gin.Engine, port string) error {
    cfg := util.GetSysConfigMirror()
    if cfg.SslEnabled && cfg.SslCertPath != "" && cfg.SslKeyPath != "" {
        if err := ginServer.RunTLS(fmt.Sprintf(":%d", cfg.SslPort), cfg.SslCertPath, cfg.SslKeyPath); err != nil {
            return err
        }
        return nil
    }
    if err := ginServer.Run(":" + port); err != nil {
        return err
    }
    return nil
}
```

需要注意:`engine.RunTLS` 不会在 cert/key 加载失败时立刻返回明确错误(`crypto/tls` 错误会打到 logger),故落库前的 `os.Stat` 校验(F5)是唯一可控、可读的失败点。

## 6. 测试策略

`router/ssl_admin_test.go`:
- 用 `httptest` + 真实登录(`/login` admin96/admin96,Token header)。
- happy:在 t.TempDir() 用 `crypto/ecdsa`(P-256)自签 `x509` → `pem.EncodeToMemory` 写 `cert.pem`/`key.pem`;POST 开启 → `code:0`;`dao.GetSystemConfig()` 校验 DB 落库(`SslEnabled==true`,`SslCertPath` 一致,`SslPort`)。
- missing-file:`enabled=true` + 指向不存在的路径 → `code:1`,不落库(再 GET 校验默认)。
- disabled-allowed:`enabled:false` + 空路径 → `code:0`(空路径 OK,因为 disabled)。
- unauthenticated: GET/POST 无 Token → handler 在 LoginHandler abort、`respCode==1`(继承既有 `salt/alert` 测试模式)。

注意:测试只驱动 handler,不真正调 `StartGin`(避免 9443 占用与 goroutine 泄漏)。

## 7. 风险与取舍

- **真正重启生效**:配置仅被 `StartGin` 冷启动读取,保存后不会热拉起 / 停掉监听。管理员需自行重启容器/进程。(文档 + CHANGELOG 提示)
- **端口漂移**:开启 HTTPS 后旧 HTTP 客户端仍可继续用 `port`(按最终实现「严格一栈」是关闭 HTTP,故旧客户端需切到 HTTPS `SslPort`;提示管理员)。
- **零值保留**:`ssl_enabled` / `ssl_port` 的平铺写入语义自洽(默认 false/9443),无需引入"***"脱敏合并。
- **cert/key 路径为敏感但不视为 secret**:仅 admin 可见,GET 原样透出;若未来要落地 RBAC 再脱敏。
