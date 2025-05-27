package cfg

import (
	"github.com/BurntSushi/toml"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
)

type Config struct {
	Listen             string `toml:"listen"`
	ApiDocHost         string `toml:"api_doc_host"`
	SteamCmdPath       string `toml:"steamcmd"`
	LogLevel           string `toml:"log_level"`
	AutoUpdateInterval int    `toml:"auto_update_interval"`
}

var G *Config

func Init(path string, defaultConfig string) error {
	G = &Config{
		Listen:       "0.0.0.0:5800",
		ApiDocHost:   "127.0.0.1:5800",
		SteamCmdPath: "steamcmd",
		LogLevel:     "warn",
	}

	if !fileutil.Exist(path) {
		err := fileutil.Write(path, []byte(defaultConfig))
		if err != nil {
			return errutil.Wrap(err)
		}
	}

	_, err := toml.DecodeFile(path, G)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}
