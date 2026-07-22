# 02 — 数据层：砍 ent + 手写 SQL + config/state 边界

**What to build:** 删除 ent ORM 及其 40+ 生成文件，替换为手写 SQL + Store struct。重新划定 config.toml（手工编辑）和 state.json（API 读写）的边界。重组 pkg 目录结构。

**Blocked by:** Ticket 01（依赖 DI 容器注入新 Store 和 config/state）

**Status:** resolved

- [x] 删除 ent/ 目录、ent 依赖，删除 pkg/database/、pkg/constant/
- [x] 在 internal/store/ 下创建 DB + Archive/Mod/Admin Store（手写 SQL）
- [x] 手写 SQL schema（4 张表：archives、mods、archive_mods、admins）+ 迁移逻辑
- [x] 实现 Get/List/Create/Update/Delete，含多对多联表查询
- [x] state.json 管理 API 可写状态（保持 Ticket 01 的 cache.State）
- [x] config.toml 只保留手工编辑字段（保持 Ticket 01 的 cfg.Config）
- [x] 删除 pkg/constant/（常量收归 main.go）
- [x] 存档/模组/管理员 CRUD 全部经 API 正常（build + test 通过）

## Comments

### 变更摘要

- 新增 `internal/store/`：DB 初始化 + 迁移 + Archive/Mod/Admin Store（手写 SQL）
- 新增 `internal/store/db.go`：WithTx 支持嵌套事务（context 传播）
- 删除 ent/ 全部 40+ 生成文件 + `pkg/database/` + `pkg/constant/`
- `go.mod`：移除 entgo.io/ent 及相关依赖
- 更新所有引用：plugin、deployer、API 层从 ent 类型切换为 store 类型
- `sql.ErrNoRows` 替代 `ent.IsNotFound`
- 所有现有测试通过
