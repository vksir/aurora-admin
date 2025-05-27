package cache

import (
	"encoding/json"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
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

var gPath string
var G *Cache

func Save() {
	content, err := json.Marshal(G)
	errutil.Check(err)
	err = fileutil.Write(gPath, content)
	errutil.Check(err)
}

func Init(path string) {
	gPath = path
	G = &Cache{
		DontStarve: cacheDontstarve{
			TickRate: 15,
		},
	}

	if !fileutil.Exist(path) {
		return
	}

	content, err := fileutil.Read(path)
	errutil.Check(err)
	err = json.Unmarshal(content, G)
	errutil.Check(err)
}
