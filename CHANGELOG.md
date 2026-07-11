# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.20260711] - 2026-07-11

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
