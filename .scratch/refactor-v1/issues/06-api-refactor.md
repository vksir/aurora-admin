# 06 — API 重构：JWT 鉴权 + v1 新路由

**What to build:** 实现 JWT 鉴权（删掉 mock auth），所有 API 切换到 `/api/v1/` 新路由体系——复数资源、console 子资源、存档 enable 改为 POST `:id/enable`、服务控制显式路由。

**Blocked by:** Ticket 03（archive 路由拆分）、Ticket 04（mod 路由）、Ticket 02（config 边界中 username/password 定义）

**Status:** resolved

- [x] config.toml 增加 username / password 字段
- [x] login 接口验密签 JWT（pkg/auth）
- [x] 所有 `/api/v1/*` 路由加 JWT 鉴权中间件
- [x] 删除 mock auth 代码（authapi 重写）
- [x] API 路由切换为 `/api/v1/` 前缀 + 复数资源名
- [x] 存档 enable 改为 `POST /api/v1/dst/archives/:id/enable`
- [x] 控制命令收到 `POST /api/v1/dst/console/` 子资源
- [x] 服务控制用显式路由，不再用 `:service/:action` 通配
- [x] 更新 assets/config.toml 默认模板
- [x] build + test 通过
- [ ] 更新 Swagger 文档（swag init）— 需手动执行

## Comments

### 变更摘要

- 新增 `pkg/auth/auth.go`：JWT 签发 + Gin 中间件
- `cfg.Config` 新增 Username/Password 字段
- `authapi` 重写：真实 login handler，签发 24h 过期 JWT
- `dstapi` 路由重写：`/api/v1/dst/` + 复数资源 + console/ + enable POST
- `svcapi` 重写：显式路由替代 :service/:action 通配
- `http.go`：/api/v1/auth（公开）+ /api/v1/*（JWT 保护）
- 所有 Swagger @Router 注解更新
