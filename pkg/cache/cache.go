package cache

import (
	"encoding/json"
	"sync"

	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
)

type cacheDontstarve struct {
	EnabledArchiveID string `json:"enabled_archive_id"`
	Token            string `json:"token"`
	TickRate         int    `json:"tick_rate"`
	PassWord         string `json:"password"`
	// LastRunBuildID 本次服务启动时对应的游戏 buildid（appmanifest），
	// 用于自动更新时检测“文件已更新但运行中进程仍旧版”
	LastRunBuildID int `json:"last_run_build_id"`
}

// RuntimeShard 记录单个分片进程的重连信息。
type RuntimeShard struct {
	PID        int    `json:"pid"`
	StartToken string `json:"start_token"`
	FIFO       string `json:"fifo"`
	Log        string `json:"log"`
}

// Runtime 记录面板启动的游戏服运行态，供面板重启后重新接管。
type Runtime struct {
	ArchID    string                  `json:"arch_id"`
	Shards    map[string]RuntimeShard `json:"shards"`
	StartedAt int64                   `json:"started_at"`
}

type Cache struct {
	DontStarve cacheDontstarve `json:"dont_starve"`
	Runtime    *Runtime        `json:"runtime,omitempty"`
}

type State struct {
	mu   sync.Mutex
	path string
	Cache
}

func Init(path string) *State {
	s := &State{
		Cache: Cache{
			DontStarve: cacheDontstarve{
				TickRate: 15,
			},
		},
	}
	s.path = path

	if !fileutil.Exist(path) {
		return s
	}

	content, err := fileutil.Read(path)
	errutil.Fatal(err)
	err = json.Unmarshal(content, &s.Cache)
	errutil.Fatal(err)
	return s
}

func (s *State) Save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveLocked()
}

func (s *State) saveLocked() {
	content, err := json.Marshal(&s.Cache)
	errutil.Fatal(err)
	err = fileutil.Write(s.path, content)
	errutil.Fatal(err)
}

// SetRuntime 持久化服务运行态（启动成功后调用）。
func (s *State) SetRuntime(rt *Runtime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Runtime = rt
	s.saveLocked()
}

// ClearRuntime 清除服务运行态（正常停止后调用）。
func (s *State) ClearRuntime() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Runtime == nil {
		return
	}
	s.Runtime = nil
	s.saveLocked()
}

// GetRuntime 返回运行态副本，避免调用方与持久化并发修改同一个 map。
func (s *State) GetRuntime() *Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Runtime == nil {
		return nil
	}
	rt := &Runtime{
		ArchID:    s.Runtime.ArchID,
		StartedAt: s.Runtime.StartedAt,
		Shards:    make(map[string]RuntimeShard, len(s.Runtime.Shards)),
	}
	for k, v := range s.Runtime.Shards {
		rt.Shards[k] = v
	}
	return rt
}
