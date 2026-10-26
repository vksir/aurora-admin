# Aurora Admin

Aurora Admin 是一个 Don't Starve Together 专用服务器管理面板：Web UI + REST API 管理游戏服务、存档、模组与远程控制。

## Language

**存档 (Archive)**：
一个 Klei 集群目录（`~/.klei/DoNotStarveTogether/Cluster_N`）连同它在面板数据库里的记录；同一时刻只有一个存档处于启用状态。
_Avoid_: Cluster、集群、世界

**分片 (Shard)**：
一个存档里由独立进程承载的世界，只有两个取值：`Master`、`Caves`。存档目录下的 `Master/`、`Caves/` 子目录与之对应。
_Avoid_: 服务器、实例、副本

**服务日志 (Service Log)**：
DST 服务端进程（各分片的 `dontstarve_dedicated_server_nullrenderer`）的 stdout/stderr，被 relay 模式重定向到 `service/dontstarve/log/dst-<shard>.log`。它是「本次运行」的实时输出，每次启动前被清空。
_Avoid_: 进程日志、实时日志、运行日志

**ServerLog.txt**：
DST 游戏自身在存档目录下写出的日志文件，由游戏进程维护、不属于面板管理的产物。面板不读它，也不要把它和服务日志混为一谈。
_Avoid_: 游戏日志、服务端日志
