package dstety

import (
	"github.com/vksir/vkiss-lib/pkg/database"
)

var MigrateModels = []any{
	&DontStarveAdmin{},

	&DontStarveArchive{},
	&DontStarveWorld{},

	&DontStarveMod{},
	&DontStarveModConfig{},
	&DontStarveModConfigOption{},

	&DontStarveModInArchive{},
	&DontStarveModConfigInArchive{},
}

var DropModels = []any{
	&DontStarveAdmin{},

	&DontStarveModConfigInArchive{},
	&DontStarveModInArchive{},

	&DontStarveModConfigOption{},
	&DontStarveModConfig{},
	&DontStarveMod{},

	&DontStarveWorld{},
	&DontStarveArchive{},
}

type DontStarveAdmin struct {
	database.Model
	Remark string `json:"remark"`
	KleiId string `gorm:"unique" json:"klei_id"`
}

type DontStarveArchive struct {
	database.Model
	Remark             string `json:"remark"`
	ClusterName        string `json:"cluster_name"`
	ClusterPassword    string `json:"cluster_password"`
	ClusterDescription string `json:"cluster_description"`
	MaxPlayers         int    `json:"max_players"`
	Pvp                bool   `json:"pvp"`

	World        []DontStarveWorld        `json:"world"`
	ModInArchive []DontStarveModInArchive `json:"mod_in_archive"`
}

const (
	WorldTypeMaster = "Master"
	WorldTypeCaves  = "Caves"
)

type DontStarveWorld struct {
	Type          string `json:"type"`
	ServerConfig  string `json:"server_config"`
	WorldOverride string `json:"world_override"`
}

type DontStarveModConfigInArchive struct {
	database.Model

	// Mod 未下载时填写，Mod 下载后将值置空
	Name string `json:"name"`
	LuaValue

	// 外键 DontStarveModInArchive
	DontStarveModInArchiveID string `json:"dont_starve_mod_in_archive_id"`

	// 外键 Belongs To
	ModConfigID string              `json:"mod_config_id" gorm:"default:null"`
	ModConfig   DontStarveModConfig `json:"mod_config"`

	// 外键 Belongs To
	ModConfigOptionID string                    `json:"mod_config_option_id" gorm:"default:null"`
	ModConfigOption   DontStarveModConfigOption `json:"mod_config_option"`
}

type DontStarveModInArchive struct {
	database.Model
	ConfigInArchive []DontStarveModConfigInArchive `json:"config_in_archive" gorm:"constraint:OnDelete:CASCADE"`

	// 外键 DontStarveArchive
	DontStarveArchiveID string `json:"dont_starve_archive_id"`

	// 外键 Belongs To
	ModID string        `json:"mod_id"`
	Mod   DontStarveMod `json:"mod"`
}

type DontStarveModConfigOption struct {
	Description string `json:"description"`
	Hover       string `json:"hover"`
	Default     bool   `json:"default"`
	LuaValue
}

type DontStarveModConfig struct {
	Name   string                      `json:"name"`
	Label  string                      `json:"label"`
	Hover  string                      `json:"hover"`
	Option []DontStarveModConfigOption `json:"option"`
}

func (m *DontStarveModConfig) FindOptionByLuaValueString(v string) (DontStarveModConfigOption, bool) {
	for i := range m.Option {
		if m.Option[i].LuaValue.String == v {
			return m.Option[i], true
		}
	}
	return DontStarveModConfigOption{}, false
}

type DontStarveMod struct {
	database.Model
	WorkshopId string `gorm:"unique;not null" json:"workshop_id"`

	// From Api
	Name        string `json:"name"`
	TimeCreated int    `json:"time_created"`
	TimeUpdated int    `json:"time_updated"`
	Image       string `json:"image"`

	// From Lua
	Description string                `json:"description"`
	Author      string                `json:"author"`
	Version     string                `json:"version"`
	Config      []DontStarveModConfig `json:"config" gorm:"constraint:OnDelete:CASCADE"`

	// Mod 是否下载
	Download bool `json:"download"`
}

func (m *DontStarveMod) FindConfigByName(n string) (DontStarveModConfig, bool) {
	for i := range m.Config {
		if m.Config[i].Name == n {
			return m.Config[i], true
		}
	}
	return DontStarveModConfig{}, false
}
