# 地址失效告警 #6 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 「地址失效告警」

## 1. 背景

README 清单第 6 项 —— 心跳同期落地后(失败 3 次后 `url.alive=false`), 需将 DOWN / UP 事件主动通知运维, 避免"URL 下线无人知"。

实现要点:
- 通过两个通道通知: **钉钉群机器人**(主) + **邮件**(备)。均可选, 通过 `system_config` 配置开关 + 凭据。
- 防抖窗口 10 分钟: 同一 URL 的 DOWN 告警 10 分钟内不重复发(防 storm)。恢复通知 10 分钟防抖同样生效。
- 阻尼收敛: DOWN 告警 10 分钟窗口; 之后如果 URL 自动通过自申请恢复(APRIORI #5)或心跳恢复, 发送恢复通知(recovery)。恢复通知也是 10 分钟 window(防 trace-blip)。心跳#4 + 自申请#5 的实现应不因告警而阻塞 ── 告警走异步 channel + 独立 worker goroutine。
- **单 worker 串行消费, 10 buffer channel, ≤2 msg/s**(防钉钉 20 msg/min 限流)。
- 配置页:后台 `/alert/get` + `/alert/update`(都 LoginHandler), 启停 ding/mail, 配 URL, SMTP 凭据, 柴油 secret 等。

## 2. 功能需求

| # | 需求 |
|---|---|
| F1 | 心跳 `UpdateUrlAlive(id, false)` 成功后向 alert channel 发 `AlertEvent{Id, Name, Path, Kind:Down}`。 |
| F2 | 心跳 `UpdateUrlAlive(id, true)`(管理员通过 `/url/enable` 走 admin-enable; 自申请到期通过 reactivate scheduler) 成功后向 channel 发 `AlertEvent{..., Kind:Up}`。 |
| F3 | alertDispatcher worker 从 channel 消费, 按 URL id+kind 防抖(最后成功告警时间 > 10 分钟前), 到点后: 读 `system_config` 取配置, 调用对应 sender(ningTalk先 + mail并行), 写 DB 更新 `last_alert_at`, `last_alert_is_down`。 |
| F4 | `system_config` 加 ~20 字段: `alert_enabled bool`, `alert_ding_enabled bool`, `alert_ding_webhook varchar`, `alert_ding_secret varchar 签名`, `alert_mail_enabled bool`, `alert_smtp_host/port/user/password/from/to varchar`, `alert_debounce_sec int default 600`, `alert_damp_failures int default 3`(复用心跳#4 的 failThreshold 体系)。 |
| F5 | `url` 表加 3 列: `last_alert_at timestamp NULL`, `last_alert_is_down bool`, `last_alert_fail_count int`(心跳在 down 路径累加, up 路径清零)。 |
| F6 | 后台管理 API `GET /alert/get` 返回当前 alert 配置(不含 password/secret 明文 --- 仅返 `true/false` 表示已配置)。 |
| F7 | 后台管理 API `POST /alert/update` 写入配置(接收 ding webhook url, secret; smtp 四件套; mail to; 各 enabled bool)。 |
| F8 | 两 sender 独立: `alert/sender_ding.go` 走 `net/http POST <webhook>` JSON, 签名 `timestamp + "\n" + secret` HMAC-SHA256 Base64 hex; `alert/sender_mail.go` 走 `net/smtp.SendMail` 发 plaintext + HTML。 |

## 3. 非功能约束

| # | 约束 |
|---|---|
| C1 | HTTP / SMTP 调用必须带超时(10s), 任何失败仅 log 不 panic, 不阻塞 channel 消费循环。 |
| C2 | 防抖窗口基于 `url.last_alert_at` 字段, 查询 + 写回原子的(单行 update)。 |
| C3 | 配置字段明文存库(通过 admin cookie 范围限定, 不支持 per-user)。 |
| C4 | 钉钉签名: 若 `alert_ding_secret` 为空, 发 plain webhook(无签名); 若不为空, 发 signed webhook(带 timestamp + sign 两 query param)。 |
| C5 | single worker + 10 buffer channel + ticker 限速 2 msg/s(每 url 处理前先 sleep 500ms)。 |

## 4. API 契约

### 心跳集成(代码内部, 无 HTTP)

心跳 `scheduler/heartbeat_scheduler.go::oneUrlBeat()` 在下线/恢复路径末尾调:
```go
// 下线: next >= failThreshold && UpdateUrlAlive(id,false) 成功后
alert.Emit(alert.Event{Id:u.Id, Path:u.Path, Kind:alert.KindDown})

// 成功: result != "timeout" 且当前 retry 探到已 down 的 URL 首次恢复时
//(即 URL alive=false → 成功探活 → 走恢复路径)
if wasDown && nowUp { alert.Emit(alert.Event{...Kind:alert.KindUp}) }
```

参考路径: 心跳#4 已修 `oneUrlBeat` 加恢复路径(本次#6 一起)。

### 后台管理 API

| 端点 | 鉴权 | Body/Query | Success(200) | Error(200 + code:1) |
|---|---|---|---|---|
| `GET /alert/get` | LoginHandler | 无 | `{code:0,data:{...config fields..., ding_secret_masked:"***", smtp_password_masked:"***"}}` | `code:1 admin only` |
| `POST /alert/update` | LoginHandler | 将所有 config fields(除 masked 外)作为 input | `{code:0,msg:"ok"}` | miss field 则维持原值(`code:0` 仍返回 success)`code:1` 仅当 parse fail |

## 5. 实现结构

### 文件清单

| 文件 | 类型 | 职责 |
|---|---|---|
| `alert/dispatcher.go` | 新 | 单 worker goroutine 消费 channel(`<-chan Event`) + ticker 限速 2/s, 读 `system_config`, 调 sender, 写 DB `last_alert_at` |
| `alert/channels.go` | 新 | `var EventCh = make(chan Event, 10)` package-level; `Emit(e Event)` try-send(drop if buffer full); `Event` struct; `Kind` enum(Up/Down) |
| `alert/sender_ding.go` | 新 | `Send(e Event, cfg entity.SystemConfig) error`: 走 `http.Client{Timeout:10s.Post(webhook, "application/json", body)` |
| `alert/sender_mail.go` | 新 | `Send(e Event, cfg entity.SystemConfig) error`: 走 `smtp.SendMail(host+port, auth, from, []string{to}, msg)` |
| `alert/sender_ding_test.go` | 新 | 测试 JSON 拼装 + 签名算法匹配 |
| `system_config_dao.go` | 改 | GetAlertConfig()读新字段; UpdateAlertConfig(newFields)更新 |
| `router/alert_admin_router.go` | 新 | RouterTemplate 暴露 `GET /alert/get` + `POST /alert/update` |
| `main.go` | 改 | RegisterRouter 后追加 `&router.AlertAdminRouter{}` |
| `scheduler/heartbeat_scheduler.go` | 改 | 引入 `alert.Emit(event)` 调用, 下线成功后 + 首次恢复成功后 |
| `entity/dao_entity.go::SystemConfig` | 改 | 加 ~6 新字段(`alert_enabled/ding_enabled/ding_webhook/ding_secret/mail_enabled/smtp_host/smtp_port/smtp_user/smtp_password/smtp_from/smtp_to/debounce_sec`) |
| `entity/dao_entity.go::Url` | 改 | 加 3 列(`last_alert_at timestamp/last_alert_is_down bool/last_alert_fail_count int`) |

## 6. 数据流

```
心跳 #4 path (UpdateUrlAlive(id, false) success)
   └─ dispatcher channel buffer (cap 10)  可能 drop if full
       └─ alertDispatcher worker (single goroutine)
            ├─ 1. read url.last_alert_at + last_alert_is_down
            │     skip if within debounce window 且 kind == last_alert_is_down
            │     skip if alert disabled 或 ding+mail both disabled
            ├─ 2. send dingTalk(json, 10s timeout, drop err log only)
            ├─ 3. send mail(smtp, 10s timeout, drop err log only)
            └─ 4. update url: last_alert_at=NOW(), last_alert_is_down=(kind==Down), fail_count++(down) 或 0(up)
```

## 7. 边界情况 + 测试

| 场景 | 处理 |
|---|---|
| alert disabled | dispatcher 读 channel 后立即 Drop, return |
| ding+mail 都 disabled | 同 |
| 10 分钟窗口内重复 down | skip(用 `url.last_alert_at > now-10min` 判定) |
| 恢复通知(kind=Up) | 防抖同 logic(字段 last_alert_is_down 区分) |
| DB write sender err | log.Printf(err), 不 panic, 下次 beat 重新 emit(likely dup, debounce 再 skip) |
| HTTP/smtp 10s 超时 | timeout 后 log + 不更新 last_alert_at, 下次 beat 重新 emit(但被 debounce 可能被 skip --- off) |
| single worker 10 buffer 满 | drop new event, log.Printf(once/sec) |
| smtp 发送 auth 失败 | log + skip, 不 double-fail |
| 钉钉签名 | HMAC-SHA256(timestamp+"\n"+secret, key=secret) → base64 → urlencode |
| 钉钉 plain webhook(secret empty) | 直接 POST, 不 timestamp/sign 参数 |

## 8. sender JSON 参考

钉钉 markdown 体:
```json
{"msgtype":"markdown","markdown":{"title":"云阶服务告警","text":"### 地址失效告警\n- URL ID: ...\n- Path: ...\n- 状态: DOWN\n- 时间: ..."}}
```

邮件体: 标准 RFC 5322 envelope(`From`, `To`, `Subject`, `text/plain` body), `%0D%0A` 换行。

## 9. 风险与备忘

- 每秒 2 msg 限速意味着极端 10 条 down URL 考虑 delay 5s 处理; 10 buffer 假定"瞬间 storm 大多 <10 URL"。
- ding 20 msg/min + 当前 debounce 10min 策略足以覆盖。
- SMTP 密码/ ding secret 明文存库, 通过 admin cookie 限定范围 — 不改写 admin multi-用户体系。
- 心跳失败到告警 emit 走 channel, 最坏情况 buffer 满 drop → 运维可能短暂 miss 事件,但 10 分钟 debounce window 内重复 emit 会补发。
- `url.last_alert_at` 原子写依赖 xorm single-row update; 并用心跳 + 多 admin 同时写 heartbeat-recovery-restore-quiet 允许。
- 恢复通知 logic 依赖 `wasDown && nowUp` 判定: 心跳成功处理后读一次 URL alive 字段看出"之前是 down 这次 up"。这是最低开销的方案(一次 DB read per recovery event)。
