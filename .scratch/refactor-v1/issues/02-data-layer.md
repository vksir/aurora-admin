# 02 — 数据层：砍 ent + 手写 SQL + config/state 边界

**What to build:** 删除 ent ORM 及其 40+ 生成文件，替换为手写 SQL + Store struct。重新划定 config.toml（手工编辑）和 state.json（API 读写）的边界。重组 pkg 目录结构。

**Blocked by:** Ticket 01（依赖 DI 容器注入新 Store 和 config/state）

**Status:** ready-for-agent

- [ ] 删除 ent/ 目录、ent 依赖，删除 pkg/database/
- [ ] 在 internal/entity/dstety/ 下创建 ArchiveStore、ModStore、AdminStore（每表一个 struct，持有 *sql.DB）
- [ ] 手写 SQL schema（4 张表：archives、mods、archive_mods、admins）+ 迁移逻辑
- [ ] 实现 Get/List/Create/Update/Delete，含多对多联表查询
- [ ] 创建 state.json 管理 API 可写状态（token、enabled_archive_id、tick_rate、password）
- [ ] config.toml 只保留手工编辑字段（listen、steamcmd_path、log_level、proxy、username、password）
- [ ] pkg 重组：删除 constant；合并 middleware+comety 为 httputil；合并 cfg/cache 职责不清的边界；拍平 util 子包（保留 errutil、fileutil）
- [ ] 存档/模组/管理员 CRUD 全部经 API 正常；config.toml 和 state.json 正确独立读写
