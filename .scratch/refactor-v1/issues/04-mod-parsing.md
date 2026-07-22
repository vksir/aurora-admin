# 04 — Mod 解析：砍 gopher-lua + 纯正则

**What to build:** 删除 gopher-lua 依赖。modoverrides.lua 的 enabled 判定用纯正则实现；mod 内部配置的美化展示推到前端处理。删除 dstety.LuaValue 类型。

**Blocked by:** Ticket 02（依赖新 entity model 做 ModStore）

**Status:** ready-for-agent

- [ ] 删除 gopher-lua 依赖（go.mod + import）
- [ ] `parseModLuaByRegex` 重写为纯正则解析（enabled 判断 + 配置块截取）
- [ ] `fetchModInfoFromLua` 及其 Lua 解释器逻辑删除
- [ ] 删除 internal/entity/dstety/ 中 LuaValue 类型及依赖 gopher-lua 的方法
- [ ] 模组列表、创建、更新、删除全部正常
- [ ] modoverrides.lua 生成内容正确（格式不变）
