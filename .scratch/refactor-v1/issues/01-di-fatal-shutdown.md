# 01 — 基础骨架：依赖注入 + 错误处理 + 优雅退出

**What to build:** 把全局变量依赖改为构造函数注入，`errutil.Check` 改为 `errutil.Fatal`（panic），HTTP server 支持 graceful shutdown。构建系统的"骨骼"——所有后续 ticket 在此之上搭建。

**Blocked by:** None — 可立即开始。

**Status:** resolved

- [x] 全局变量（`cfg.G`、`cache.G`、`database.G`）改为构造函数注入
- [x] application struct 统一持有并传递所有依赖
- [x] `errutil.Check` → `errutil.Fatal`：内部用 panic，main 层 recover 打印后退出
- [x] Fatal 仅限 Init 阶段，API handler 链路 return error
- [x] 删除 `application.ctx`，HTTP server 实现 graceful shutdown（SIGTERM → srv.Shutdown）
- [x] `go build` + 启动 + 访问页面 + Ctrl+C 优雅退出

## Comments

### 变更摘要

- `pkg/util/errutil/errtuil.go` → `errors.go`：`Check` 改名为 `Fatal`，`os.Exit(1)` 改为 `panic`
- `main.go`：`serve()` 中 `defer recover` 捕获 Fatal panic；`cfg.Init`/`cache.Init`/`database.Init` 接收返回值
- `internal/app/app.go`：删除 `ctx` 字段；`Run()` 接收 `*cfg.Config`、`*cache.State`、`*ent.Client`；HTTP server graceful shutdown
- `internal/app/http.go`：`serveHttp` 返回 `*http.Server`，不再调用 `e.Run()` 阻塞
- `pkg/cfg/cfg.go`：`Init` 返回 `(*Config, error)`，删除全局 `G`
- `pkg/cache/cache.go`：`Init` 返回 `*State`（嵌入 `Cache`），`Save` 改为方法，删除全局 `G`
- `pkg/database/database.go`：`Init` 返回 `*ent.Client`，删除全局 `G`
- `pkg/util/util.go`：`ProxyClient` 改为接收 proxy 参数
- `internal/plugin/dontstarve/*`：Plugin 和 deployer 持有 `*cfg.Config` 和 `*cache.State`，替代全局变量访问
- 所有测试通过
