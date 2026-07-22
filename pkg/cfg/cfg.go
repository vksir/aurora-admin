package cfg

import (
	"github.com/BurntSushi/toml"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
)

type Config struct {
	Listen             string `toml:"listen"`
	ApiDocHost         string `toml:"api_doc_host"`
	SteamCmdPath       string `toml:"steamcmd"`
	LogLevel           string `toml:"log_level"`
	AutoUpdateInterval int    `toml:"auto_update_interval"`
	Proxy              string `toml:"proxy"`
	Username           string `toml:"username"`
	Password           string `toml:"password"`
}

func Init(path string, defaultConfig string) (*Config, error) {
	c := &Config{
		Listen:       "0.0.0.0:5800",
		ApiDocHost:   "127.0.0.1:5800",
		SteamCmdPath: "steamcmd",
		LogLevel:     "warn",
		Proxy:        "",
		Username:     "admin",
		Password:     "admin",
	}

	if !fileutil.Exist(path) {
		err := fileutil.Write(path, []byte(defaultConfig))
		if err != nil {
			return nil, errutil.Wrap(err)
		}
	}

	_, err := toml.DecodeFile(path, c)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return c, nil
}
