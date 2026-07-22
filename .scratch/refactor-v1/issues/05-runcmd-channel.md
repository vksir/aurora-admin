# 05 — RunCmd：channel 同步替代 sleep + regex

**What to build:** SubProcess 内部增加 pending command channel。向游戏服发命令时，通过 channel 匹配请求和响应，替代 `time.Sleep(500ms)` 硬等待和 `BEGIN_CMD` 标记字符串 + 正则截取。

**Blocked by:** None — 和其他 ticket 独立。

**Status:** ready-for-agent

- [ ] SubProcess 内部新增 pending command channel 和请求-响应匹配逻辑
- [ ] RunCmd 改为通过 channel 发送命令、等待结果
- [ ] 删除 `time.Sleep(500ms)` 硬等待
- [ ] 删除 `[BEGIN_CMD]...[END_CMD]` 标记字符串和正则截取
- [ ] `GetPlayers` 返回正确玩家列表
- [ ] `Announce` 公告成功发送
- [ ] `Regenerate` / `Rollback` 命令正确执行
