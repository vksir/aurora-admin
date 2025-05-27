package dstety

import (
	"aurora-admin/internal/entity/comety"
)

type DontStarveAdmin struct {
	comety.Model
	Remark string `json:"remark"`
	KleiId string `gorm:"unique" json:"klei_id"`
	Enable bool   `json:"enable"`
}

type DontStarveArchive struct {
	comety.Model
	ClusterName        string `json:"cluster_name"`
	ClusterPassword    string `json:"cluster_password"`
	ClusterDescription string `json:"cluster_description"`
	MaxPlayers         string `json:"max_players"`
	Pvp                string `json:"pvp"`

	Worlds []DontStarveWorld        `json:"worlds"`
	Mods   []DontStarveModInArchive `json:"mods"`
}

const (
	DontStarveWorldTypeMaster = "Master"
	DontStarveWorldTypeCaves  = "Caves"
)

type DontStarveWorld struct {
	comety.Model
	Type          string `json:"type"`
	ServerConfig  string `json:"server_config"`
	WorldOverride string `json:"world_override"`

	// 外键
	DontStarveArchiveID string `json:"dont_starve_archive_id"`
}

type DontStarveModInArchive struct {
	comety.Model
	CurrentConfig string `json:"current_config"`

	// 外键
	DontStarveArchiveID string `json:"dont_starve_archive_id"`

	// 外键 Belongs To
	ModID string        `json:"mod_id"`
	Mod   DontStarveMod `json:"mod"`
}

type DontStarveMod struct {
	comety.Model
	WorkshopId string `gorm:"unique" json:"workshop_id"`
	Name       string `json:"name"`
	Config     string `json:"config"`
}

func (d *DontStarveMod) ToInArchive(archId string) DontStarveModInArchive {
	return DontStarveModInArchive{
		CurrentConfig: d.Config,
		ModID:         d.ID,
	}
}
