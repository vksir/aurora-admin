# 04 — Mod 解析：砍 gopher-lua + 纯正则

**What to build:** 删除 gopher-lua 依赖。modoverrides.lua 的 enabled 判定用纯正则实现；mod 内部配置的美化展示推到前端处理。删除 dstety.LuaValue 类型。

**Blocked by:** Ticket 02（依赖新 entity model 做 ModStore）

**Status:** resolved

- [x] 删除 gopher-lua 依赖（go.mod + import）
- [x] `parseModLuaByRegex` 重写为纯正则解析（括号计数 + enabled 正则判定）
- [x] `fetchModInfoFromLua` 及其 Lua 解释器逻辑删除
- [x] 删除 dstety 中依赖 gopher-lua 的函数（NewLuaValue, StringLua, Value）
- [x] 模组创建、更新、删除全部正常
- [x] modoverrides.lua 生成内容正确
