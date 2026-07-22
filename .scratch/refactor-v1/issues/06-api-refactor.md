# 06 — API 重构：JWT 鉴权 + v1 新路由

**What to build:** 实现 JWT 鉴权（删掉 mock auth），所有 API 切换到 `/api/v1/` 新路由体系——复数资源、console 子资源、存档 enable 改为 POST `:id/enable`、服务控制显式路由。

**Blocked by:** Ticket 03（archive 路由拆分）、Ticket 04（mod 路由）、Ticket 02（config 边界中 username/password 定义）

**Status:** ready-for-agent

- [ ] config.toml 增加 username / password 字段
- [ ] login 接口验密签 JWT
- [ ] 所有 `/api/v1/*` 路由加 JWT 鉴权中间件
- [ ] 删除 mock auth 代码
- [ ] API 路由切换为 `/api/v1/` 前缀 + 复数资源名
- [ ] 存档 enable 改为 `POST /api/v1/dst/archives/:id/enable`，删除 archive/enable 独立路由子树
- [ ] 控制命令收到 `POST /api/v1/dst/console/` 子资源
- [ ] 服务控制用显式路由，不再用 `:service/:action` 通配
- [ ] 更新 Swagger 文档（swag init）
- [ ] 新路由全量 200；未认证请求 401
