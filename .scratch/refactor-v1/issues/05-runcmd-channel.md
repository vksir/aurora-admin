# 05 — RunCmd：channel 同步替代 sleep + regex

**What to build:** SubProcess 内部增加 SendCommand 方法，通过临时 output collector + channel + timeout 替代 `time.Sleep(500ms)` 和 `[BEGIN_CMD]...[END_CMD]` 标记。

**Blocked by:** None — 和其他 ticket 独立。

**Status:** resolved

- [x] SubProcess 新增 `SendCommand` 方法，使用临时 output collector + timer timeout
- [x] RunCmd 改为 delegate 到 `process.SendCommand(cmd, DefaultCommandTimeout)`
- [x] 删除 `time.Sleep(500ms)` 硬等待
- [x] 删除 `[BEGIN_CMD]...[END_CMD]` 标记字符串和正则截取
- [x] 删除 plugin.go 中未使用的 `regexp`、`strings` import
- [x] `GetPlayers`/`Announce`/`Regenerate`/`Rollback` 通过 RunCmd 自动受益
