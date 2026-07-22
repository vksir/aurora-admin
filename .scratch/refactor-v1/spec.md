Status: ready-for-agent

# Refactor v1: 简化架构、清理年久失修的代码

## Problem Statement

Aurora Admin 是一个 Don't Starve Together 专用服务器管理工具，提供 Web UI + REST API 进行游戏服务管理、存档管理、模组管理和远程控制。当前代码库面临以下问题：

1. **过度抽象**：`pkg/service` 实现完整的状态机 + 子进程管理 + 通用接口，但全项目只有一个实现（DST）。`pkg/registry` 搭建了 pub-sub + cron 框架，只服务一个定时任务。
2. **重量级依赖**：ent ORM 为 3 张简单表生成了 40+ 文件；gopher-lua 完整 Lua 运行时仅用于解析 mod 配置中的 enabled 字段。
3. **全局变量满天飞**：`cfg.G`、`cache.G`、`database.G`、`service.gService` 散布在各处，无法并行测试，依赖关系不可见。
4. **配置与状态边界模糊**：Token、TickRate、Password 和 EnabledArchiveID 混在 JSON cache 文件里，Listen、SteamCmdPath 在 TOML 里，一次"配置存档"操作横跨三种存储。
5. **死代码和杂物**：`thirdparty/github/` 零引用、`scripts/` 含 Terraria shell 脚本、`assets/tmodloader/` 无对应 plugin。
6. **API 设计混乱**：`archive/enable` 路由重复、控制命令与 CRUD 混排、单复数不一致、DELETE 带 JSON body、mock 鉴权无实际保护。
7. **技术债务**：pathHelper 三层继承只做路径拼接、RunCmd 用 `time.Sleep(500ms)` + 正则截取、`errutil.Check` 用 `os.Exit(1)` 跳过 defer、`application.ctx` 存而不用导致无 graceful shutdown。

作为一个未发布的应用，现在是修正这些问题的窗口期。核心原则：**一切从简，简单就是好**。

## Solution

对 Aurora Admin 进行一次系统性重构，目标是：砍掉不必要的抽象和依赖，统一数据存储边界，清理死代码，重新设计 API，建立可测试的架构。重构后代码量预计缩减 40-50%，依赖数减少 30%，所有 HTTP API 可通过集成测试验证。

## User Stories

### 代码架构

1. As a 新加入的开发者，我想一眼看明白"启动服务"干了什么，而不是跳三层抽象才能理解。
2. As a 项目维护者，我想删掉 40+ 个 ent 生成文件，用手写 SQL 替代，减少认知负担。
3. As a 开发者，我想看构造函数签名就知道一个结构体的依赖，而不是到处搜全局变量引用。
4. As a 测试编写者，我想能并行跑测试而不互相踩全局状态。

### 存储与配置

5. As a 运维，我想手动编辑 config.toml 设置监听地址和 steamcmd 路径，有注释和易懂的格式。
6. As a 前端用户，我想通过 Web UI 修改 Token、密码、TickRate，改动持久化。
7. As a 运维，我想清楚区分"手工编辑的配置"和"API 管理的状态"，不相互覆盖。
8. As a 开发者，我想用一个 JSON 文件管理所有 API 可写的运行时状态，不需要理解 TOML 解析细节。

### API 设计

9. As a 前端开发者，我想 API 路由一致遵循 REST 惯例（复数资源、标准动词），减少集成困惑。
10. As a 外网用户，我想服务有真正的鉴权保护，不是 mock token 形同虚设。
11. As a 前端开发者，我想区分"数据操作"和"控制命令"，前者走 RESTful CRUD，后者走统一的 console 子资源。
12. As a 运维，我想通过 API 启用某个存档，不需要记住两套不同的 upload 路由。

### 代码清理

13. As a 开发者，我不想在一堆未使用的死代码中搜索关键逻辑。
14. As a 开发者，我想路径拼接是直观的函数调用，不是三层 struct 继承。

### 进程交互

15. As a 开发者，我想向游戏服发命令后可靠地拿到结果，而不是靠 `time.Sleep(500ms)` 碰运气。
16. As a 运维，我想服务收到 SIGTERM 后优雅关闭 HTTP server，让进行中的请求处理完。

### 测试

17. As a 开发者，我想写 HTTP 集成测试验证整个链路（router → handler → DB → response），一个 seam 覆盖全栈。

## Implementation Decisions

### 架构决策

1. **保留 `pkg/service` 抽象层，不扩展。** 当前 Interface 支持 DST 的进程模型，添加新游戏时若不适配则允许绕过，不强行适配。等第二个游戏实现后再重新审视 Interface 形状。

2. **保留 `pkg/registry`（pub-sub + cron）。** 不砍掉，尽管当前只有一个 cron job 使用。

3. **保留嵌入式前端。** Vue SPA build 产物通过 `//go:embed` 嵌入 Go 二进制，维持单文件部署。

### 数据层决策

4. **砍掉 ent ORM。** 换用 `modernc.org/sqlite`（已在依赖中） + 手写 SQL。三张表（archive、mod、admin）每表一个 Store struct，方法 Get/List/Create/Update/Delete 直接写 SQL。JSON 字段（`world`）用 `json.Marshal/Unmarshal` 序列化。多对多关联（archive ↔ mod）加一张 `archive_mods` 中间表，两条 SQL 实现联查。

5. **Model 定义放 `internal/entity/dstety/`**，每个 entity 一个文件，包含 struct 定义 + Store 方法。不做泛型 Repository 或 query builder。

6. **config.toml（手工编辑）和 state.json（API 读写）按编辑者划界。**
   - config.toml：`listen`、`steamcmd_path`、`log_level`、`proxy`、`username`、`password`
   - state.json：`token`、`enabled_archive_id`、`tick_rate`、`password`（服务器密码）

7. **合并 cfg 和 cache 为统一的 store 包**，一个 JSON 文件管理 API 可写状态。启动时一把读完。

### 依赖注入决策

8. **删除所有包级全局变量**（`cfg.G`、`cache.G`、`database.G`、`service.gService`、`log.logger` 保留一个默认实例）。所有依赖通过构造函数注入，由 `application` struct 统一持有并传递。

### 路径管理决策

9. **砍掉 pathHelper 三层 struct 继承**（pathHelper → clusterPathHelper → shardPathHelper），改为纯函数。路径拼接逻辑不变，只改组织方式。

### 文件组织决策

10. **pkg 按功能域重组**：
    - `config.toml` 手工配置 → 保留独立 cfg 包读 TOML
    - 运行时状态 → store 包（融合原 cache）
    - HTTP 通用工具 → httputil 包（融合 middleware + comety 响应格式）
    - 删除 constant 包（常量各归各主）
    - 删除 database 包（随 ent 一起走）
    - 保留 errutil、fileutil、log、steam

11. **`internal/plugin/steamcmd/` 删除**，路径函数移到 `pkg/steam/`。steamcmd 自身安装/卸载逻辑也放 `pkg/steam/`。

12. **`plugin_archive.go` 拆为三个文件**：archive_reader.go（从磁盘读存档 → entity）、archive_writer.go（从 entity 写磁盘，原 deployer 逻辑）、archive_transfer.go（上传下载 zip 操作）。plugin_archive.go 保留为 API 入口薄层。

13. **`pkg/util` 拍平**：保留 `errutil` 和 `fileutil` 子包，其余（convutil、util.go 的 UpdateIfChanged/ProxyClient）合并到 util 根包。

### 代码清理决策

14. **删除死代码**：`thirdparty/github/`、`scripts/`（保留 setup.sh）、`assets/tmodloader/`、`prompt/`（保留 prompt/aurora-admin-web.md）。

15. **`errutil.Check` 改为 `errutil.Fatal`，用 `panic` 替代 `os.Exit(1)`**，main 层统一 recover 打印后退出。Fatal 仅限 Init 阶段使用，API handler 链路的 I/O 操作 return error。

### 进程交互决策

16. **RunCmd 重新设计为请求-响应模型**：SubProcess 维护 pending command channel，loopOutput 匹配命令和响应，替代 `time.Sleep(500ms)` + 正则截取。

17. **HTTP server 加 graceful shutdown**：删除 `application.ctx`，监听 SIGTERM 后 `srv.Shutdown(ctx)` 等现有请求处理完。

### Mod 解析决策

18. **砍掉 gopher-lua 依赖**。modoverrides.lua 的 enabled 字段用纯正则判断，mod 内部配置美化展示推到前端处理。

### 鉴权决策

19. **从 mock 改为 JWT 鉴权**。config.toml 加 `username` / `password`，login 验密签 JWT，所有 `/api/*` 加 middleware 验 JWT。不建用户表（短期单管理员），架构预留切换到 DB 用户表的能力。

### API 设计决策

20. **重新设计 API 路由**，核心改动：
    - 加 `/api/v1/` 版本前缀
    - 所有资源用复数（archives、mods、admins、players）
    - 存档 enable 改为 `POST /api/v1/dst/archives/:id/enable`，删除 archive/enable 独立路由子树
    - 控制命令收到 `POST /api/v1/dst/console/*`，与 CRUD 分离
    - 服务控制用显式路由 `POST /api/v1/dst/service/{start,stop,restart,install,uninstall,update}`，不再用 `:service/:action` 通配
    - DELETE 单资源用路径参数 `DELETE /api/v1/dst/mods/:id`，批量删除留待后续

完整 API 契约：

```
# 认证
POST   /api/v1/auth/login
GET    /api/v1/auth/codes
GET    /api/v1/user/info

# DST 配置
GET    /api/v1/dst/config
PUT    /api/v1/dst/config

# DST 存档
GET    /api/v1/dst/archives
POST   /api/v1/dst/archives
GET    /api/v1/dst/archives/:id
PUT    /api/v1/dst/archives/:id
DELETE /api/v1/dst/archives/:id

# DST 存档传输
POST   /api/v1/dst/archives/upload
PUT    /api/v1/dst/archives/:id/upload
GET    /api/v1/dst/archives/:id/download

# DST 存档启用
POST   /api/v1/dst/archives/:id/enable
GET    /api/v1/dst/archives?enabled=true

# DST 模组
GET    /api/v1/dst/mods
GET    /api/v1/dst/mods/:id
PUT    /api/v1/dst/mods/:id
DELETE /api/v1/dst/mods/:id

# DST 管理员
GET    /api/v1/dst/admins
POST   /api/v1/dst/admins
DELETE /api/v1/dst/admins/:id

# DST 控制命令
POST   /api/v1/dst/console/players
POST   /api/v1/dst/console/announce
POST   /api/v1/dst/console/regenerate
POST   /api/v1/dst/console/rollback

# DST 服务控制
POST   /api/v1/dst/service/start
POST   /api/v1/dst/service/stop
POST   /api/v1/dst/service/restart
POST   /api/v1/dst/service/install
POST   /api/v1/dst/service/uninstall
POST   /api/v1/dst/service/update
GET    /api/v1/dst/service/status
```

### 数据库 Schema

砍掉 ent 后，手工维护 schema：

```sql
CREATE TABLE IF NOT EXISTS dont_starve_archives (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    remark TEXT,
    cluster_name TEXT,
    cluster_password TEXT,
    cluster_description TEXT,
    max_players INTEGER,
    pvp INTEGER,
    world TEXT  -- JSON: []DontStarveWorld
);

CREATE TABLE IF NOT EXISTS dont_starve_mods (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    workshop_id TEXT NOT NULL,
    name TEXT,
    description TEXT,
    image TEXT,
    time_created INTEGER,
    time_updated INTEGER,
    raw_config TEXT
);

CREATE TABLE IF NOT EXISTS archive_mods (
    archive_id TEXT NOT NULL,
    mod_id TEXT NOT NULL,
    PRIMARY KEY (archive_id, mod_id),
    FOREIGN KEY (archive_id) REFERENCES dont_starve_archives(id) ON DELETE CASCADE,
    FOREIGN KEY (mod_id) REFERENCES dont_starve_mods(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dont_starve_admins (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    name TEXT,
    klei_id TEXT NOT NULL
);
```

## Testing Decisions

### 测试哲学

好测试只验证外部可观察行为，不绑定实现细节。优先通过 HTTP 集成测试覆盖全栈，Store 层测试精细验证 SQL 正确性。

### 测试 Seam

| 层级 | Seam | 策略 |
|------|------|------|
| **HTTP（主 seam）** | Gin test router + SQLite `:memory:` | `httptest` 发请求，断言 status + body。覆盖 handler → Store → DB 全链路 |
| Store 层（辅 seam） | `*sql.DB` 指向 `:memory:` | 直接调 Store 方法，验证 CRUD + 联表查询 |
| Plugin 业务逻辑 | 注入 mock subprocess / mock steamcmd | 测存档解析、配置生成、mod 解析、命令发送 |
| 服务状态机 | mock `service.Interface` | 测状态转移、并发锁、错误路径 |

### 测试范围

- **HTTP 层**：每个 endpoint 至少一条 happy-path 测试（200 + 正确 JSON 结构）
- **Store 层**：每个方法一条测试，外加联表查询、事务回滚
- **Plugin 层**：存档解析（从磁盘读 → entity）、config 模板生成、mod 解析（正则匹配）
- **现有测试保留**：`deploy_test.go`、`plugin_mod_test.go`、`steamapi_test.go`、`steamdb_test.go`、`log_test.go` 相应调整以适应重构后结构

### 不测什么

- 不测第三方库行为（Gin、SQLite driver、slog）
- 不测模板渲染细节（验证生成文件存在 + 关键内容即可）
- 不测 steamcmd 真实网络调用（保持 mock）

## Out of Scope

- **前端改动**：前端代码不在本次重构范围，仅调整 API 路由后适配接口调用
- **新增功能**：不添加新游戏支持、不添加新管理功能
- **Docker 部署优化**：现有 Dockerfile/部署流程不变
- **性能优化**：不涉及性能瓶颈排查或优化
- **多用户系统**：鉴权仅实现单用户 JWT，不做用户管理 UI 和权限系统
- **TModLoader/Terraria plugin**：不实现，相关资产文件删除
- **日志系统重构**：`pkg/log` 保持现状
- **Steam Web API 客户端重构**：`pkg/steam/steamapi.go` 保持现状

## Further Notes

- 重构应分阶段执行：基础骨架修正 → 数据层替换 → API 重构 → 清理收尾。每阶段独立可验证。
- 砍 ent 是最大风险点，建议在独立分支上操作，合并前手动回归所有存档/模组/管理员 CRUD 操作。
- API 变更需要在重构完成后更新 Swagger 文档（`swag init`）。
- 前端 `prompt/aurora-admin-web.md` 中的 API 文档引用需要同步更新为新路由。
