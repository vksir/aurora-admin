package cache

import (
	"encoding/json"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
)

type cacheDontstarve struct {
	EnabledArchiveID string `json:"enabled_archive_id"`
	Token            string `json:"token"`
	TickRate         int    `json:"tick_rate"`
	PassWord         string `json:"password"`
}

type Cache struct {
	DontStarve cacheDontstarve `json:"dont_starve"`
}

type State struct {
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
	content, err := json.Marshal(&s.Cache)
	errutil.Fatal(err)
	err = fileutil.Write(s.path, content)
	errutil.Fatal(err)
}
