# 自动启停地址(心跳检查) 设计

> 状态: 已确认  ·  创建: 2026-07-12  ·  来源: README 「开发进度 · 自动生效或失效地址(心跳检查)」

## 1. 背景

README 开发进度清单第 4 项「自动生效或失效地址(心跳检查)」。

现有能力:
- `url` 表已有 `Alive bool` + `Retry int`(`entity/dao_entity.go::Url`)。
- `util/ping_util.go::GetTCPSpeed(url, ch)` 做 500ms TCP 探活(探活失败返 `"timeout"`)。
- scheduler 已有 180s hot-clean 调度器(`scheduler/hot_time_clean_scheduler.go`), 模式是 `tm.ScheduleFunc(dur, fn)`。
- 本次同期已修 `dao.UpdateUrlAlive(id, bool) bool` 写 DB+cache 双持(本次 `feat(dao)`)。

本次目标: 引入 60s 心跳节拍, 对每个 `Alive=true` 的 URL 做 TCP 探活, 失败累加 `url.Retry+1`, 达 N=3 时执行 `dao.UpdateUrlAlive(id, false)`。

## 2. 功能 / 非功能需求

### 功能需求

| # | 需求 |
|---|---|
| F1 | 每 60s 对所有 `Alive=true` 的 URL 做 TCP 探活 |
| F2 | 探活失败 → `url.Retry+1` 写 DB(内存中不单独持 map) |
| F3 | `url.Retry >= 3` 时 `dao.UpdateUrlAlive(id, false)` 下线 |
| F4 | 探活成功 → `url.Retry=0`(只要 retry!=0 才写, 省 IO) |
| F5 | `url.Alive=false`(admin 手动 or 心跳自身所致)→ 跳过(不探不加) |
| F6 | 心跳只摘除; 不下线的长期恢复需要 admin 手动 enable(不会自动复活) |
| F7 | admin-enable 通过同 `UpdateUrlAlive(id, true)` 执行; 该写入会 `Retry=0` 解锁再次累加 |
| F8 | 失败累加 + 下线操作有 stderr log, admin 可查阅 |

### 非功能约束

| # | 约束 |
|---|---|
| C1 | 复用现有 `url.Retry` 字段, 不引入新 schema |
| C2 | 探活并发执行(URL 数 ≤ 百级时单次 beat 不会跨节拍) |
| C3 | beat 迅速退出; 全量为空快速 return |
| C4 | 本次不涉及 system_config 配置; 配置(节拍/阈值)写成常量; 未来独立 PR 再 outsourced 到 sysconfig |

### 显式 YAGNI (本次不做)

- 不做告警(#6 告警功能单列)。
- 不做"长期停用"标记,心跳只摘 revive admin。
- 不做 HTTP-mode 探活(仅 TCP)。
- 不做在 system_config 加开关先默认 heartbeat on; future PR 加 `heartbeatEnabled`。

## 3. API 契约

**无新路由 / 新 API / 新 UI 字段**。

唯一对外表现:
- stderr 输出: `[heartbeat] url id=<id> path=<p> fail (retry=<n>/<N>)` (累加) 与 `[heartbeat] url id=<id> path=<p> set ALIVE=false after N failures` (下线)。
- `/slist` 列表回返的 `retry` 字段已透出 admin 可看(`entity.Url.Retry`)。

## 4. 数据流 + 代码位置

| 改动文件 | 类型 | 职责 |
|---|---|---|
| `scheduler/heartbeat_scheduler.go` | 新 | `beat()` + `registerHeartbeatScheduler(tm)` 单文件 |
| `scheduler/scheduler_register.go` | 改 | 加 `registerHeartbeatScheduler(tm)` 调用 |
| `dao/url_dao.go` | 改 | 新增 `IncrementUrlRetry(id, delta) bool`(-1 表清零) |
| `util/ping_util.go` | 不变 | 复用 `GetTCPSpeed` |
| `collection/proxy_mode.go`/`self_help_mode.go` | 不变 | 心跳只读 cache、不改 |

### 心跳核心循环(`heartbeat_scheduler.go::beat()`)

```go
func beat() {
    // 1. 整包读 cache 快照
    collection.MWorkCllection.RLock()
    all := make([]entity.Url, 0)
    for _, urls := range collection.WorkCllection {
        all = append(all, urls...)
    }
    collection.MWorkCllection.RUnlock()

    const failThreshold = 3

    var wg sync.WaitGroup
    for i := range all {
        u := all[i]
        if !u.Alive { continue }   // 跳过 admin/heartbeat 已停的(含 admin-manual 禁用)

        wg.Add(1)
        go func(u entity.Url) {
            defer wg.Done()
            ch := make(chan string, 1)
            go util.GetTCPSpeed(u.Path, ch)
            result := <-ch

            if result == "timeout" {
                // 累加: 失败 retry+1
                newRetry := currentRetry(u.Id) + 1   // 实现者确认用 session.Incr 还是 SELECT+UPDATE
                if newRetry >= failThreshold {
                    dao.UpdateUrlAlive(u.Id, false)  // 本次同期已修
                    log.Printf("[heartbeat] url id=%d path=%s set ALIVE=false after %d failures", u.Id, u.Path, newRetry)
                } else {
                    log.Printf("[heartbeat] url id=%d path=%s fail (%d/%d)", u.Id, u.Path, newRetry, failThreshold)
                }
            } else {
                // 成功: retry 清零(前提是 retry!=0 才写)
                if currentRetry(u.Id) != 0 {
                    dao.UpdateUrlRetry(u.Id, 0)
                }
            }
        }(u)
    }
    wg.Wait()
}
```

**关键实现提示**:
- `currentRetry(id)`:小表下 SELECT url WHERE id=n 取 Retry 字段即可;但本次采用推荐法 **SELECT + UPDATE**(见下方)。
  - xorm 1.4 带 `Incr`/`Decr` 语义,我们**不**用作本次, 因为:
    - 失败/成功两条路径语义不同(失败是 +1,成功是归零), `Incr` 只能做 "+1" 一种动作, 成功归零走 `Incr(..., -r)` 需要预先读 r, 等义。
    - 用 SELECT + UPDATE 语义直笔实现者一望即明, 出错也好调试。
  - 最终实施法:`Read + Write`:
    ```go
    func IncrementUrlRetry(id int, targetRetry int) bool {
        // targetRetry: (>=0) 目标 retry 值; 调用方: 失败时传 currentRetry(id)+1; 成功时传 0
        var u entity.Url
        has, err := PublicEngine.ID(id).Get(&u)
        if err != nil || !has { return false }
        if u.Retry == targetRetry { return true } // 已经相等,省 IO
        u.Retry = targetRetry
        affected, err := PublicEngine.ID(id).Cols("retry").Update(&u)
        return err == nil && affected == 1
    }
    ```
    这同一函数可服「失败累加」与「成功归零」两种语义 — 调用方传 targetRetry。
- `snapshot`:整拍在 RLock 内拷出后立即 unlock, 心跳 goroutine 不持有 cache 锁, 故不影响热路径。
- 并发控制:`single-goroutine-per-url + wg.Wait()`; URL 数 ≤ 百级时并发探活单拍内完成, 与 60s 节拍嵌套安全。

## 5. 错误处理 + 边界

| 场景 | 处理 |
|---|---|
| URL 数=0 时 beat | snapshot 为空 → for 循环不进入 → return, 不 DB 写 |
| URL 刚好在 beat 正管理员 `disable` | 双写竞争: admin disable 写 Alive=false + Retry=0 与 heartbeat retry+1 在 xorm 单行原子; 最后一次写胜, 无逻辑错误 |
| URL 刚好在 beat 正管理员 `enable` | admin enable 写 Alive=true + Retry=0; 心跳同时累加一个新 retry; 下个 beat 探活成功 retry 清零; 需要极简 3 次失败才 down; 总体稳定 |
| beat 中途被 SIGKILL | beat 每周期独立, drop 的 URL 下一拍补探 |
| log + 最近 down 日志 | log 包默认 stderr, GIN release mode 下也写; 不错 |
| `currentRetry` 的 SQL 跑法 | xorm 1.4+ 支持 `session.ID(id).Incr("retry", ±1)`; 若不支持, 回退到 `SELECT retry FROM url WHERE id=?` + `session.ID(id).Cols("retry").Update(&Url{Retry: n})` |
| 探活 proxy target(如 `http://127.0.0.1:xxxx`) | address 已经在 `WorkCllection.Path` 里, 心跳直接用 `util.GetTCPSpeed(u.Path, ch)`; 无需解析 |

### URL cache 外泄风险

snapshot 在 `MWorkCllection.RLock()` 保护下拷出, 心跳 goroutine 拿到的局部 `u Path` 字段读时 cache 已被其他写 admin disable/enable 改过, 无问题(探活针对当时 snapshot 快照, 本拍结果对应当时状态)。

## 6. 测试

1. `scheduler/heartbeat_scheduler_test.go`: 测试写一个 fixture:
   - 用 temp sqlite 文件(`t.TempDir()+ "/test.db"`)作为 `xorm.NewEngine("sqlite3", path)` 替代主 DB, 但更简单做法是复用 dev DB `cloud_step.db` 中已有的测试 url 并在 tearDown 恢复(url.Retry / url.Alive 写回原值), 不引入跨测试 DB 工具。
   - Seed 一组 `Alive=true` URL, 跑一轮 beat, 断言 DB `url.Retry` 正确累加(失败 URL) / 仍为 0(成功 URL)。
   - Seed 一组刚好 retry=N-1 的 URL, 模拟连续失败 1 次, 断言 beat 后 `url.Alive=false`。
2. 实际 sanity 启动二进制: `go run main.go 9100` + grep stderr 看 heartbeat log。

## 7. 风险与备忘

- **节拍严格以 60s 起**: 如果 beat goroutine 跑满 59s(极限百 URL × 500ms+), 下一拍立即开启(antlabs timer.ScheduleFunc 在 fn 返回后再 schedule 下一拍), 长期会有所偏移, 但 URL 数通常两位数, 不打节拍。
- **README 字面偏离**: README 写「检查三次无效则**发送告警且长期停用**」, 本次 spec 拆为 #4(心跳只摘除) + #6(告警单列)。即: 心跳**无"长期停用"标记**(admin-enable 即 Retry=0 解锁, 无需 long-term flag),**无即时告警**(事件在 stderr log, hook point 留给 #6)。这是架构分工而非遗漏。与用户定案一致:「 心跳只摘除 + admin 负责 6 警」联动。
- admin-enable 与 heartbeat 的联动:"admin-enable 即 Retry=0", 下次 beat 如果 URL 未恢复已无 buffer 立即累加。如果 URLs 频繁 flip, 建议在 #4 落地后做"跳过刚刚 admin-enable 的第一个 beat"的保护 — 当前范围不做,留给 #6 告警时一齐评估。
- **C 路径未来**: 后续引入 `system_config.heartbeatEnabled` + `heartbeatIntervalSec` 等配置开关, 需要重启后生效的常量改成 `GetSysConfigMirror()` 已查。schema 扩在 sysconfig + 新增 config 字段; 心跳 cadence 常量改成 `time.Duration(interval) * time.Second`。不做在这一 PR。
