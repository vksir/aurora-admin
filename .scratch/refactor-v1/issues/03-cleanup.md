# 03 — 代码清理：路径函数 + 死代码 + steamcmd 归位 + archive 拆分

**What to build:** 把 pathHelper 三层继承替换为纯函数；删除所有死代码和杂物；steamcmd 逻辑归入 pkg/steam；plugin_archive.go 拆为 reader/writer/transfer 三个文件。

**Blocked by:** Ticket 02（依赖新 entity model 做 archive 的 reader/writer 拆分）

**Status:** resolved (merged with Ticket 04 commit)

- [x] pathHelper / clusterPathHelper / shardPathHelper 三层 struct → 纯函数
- [x] 删除 thirdparty/github/（零引用）
- [x] 删除 scripts/（保留 setup.sh）
- [x] 删除 assets/tmodloader/
- [x] internal/plugin/steamcmd/ 路径函数归入 pkg/steam/workshop.go
- [x] `go build` 无未引用包；存档上传/下载/创建正常
