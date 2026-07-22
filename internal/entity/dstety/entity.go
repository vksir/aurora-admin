package dstety

type Status struct {
	Status int `json:"status"`
}

type Config struct {
	EnabledArchiveId string `json:"enabled_archive_id"`
	Token            string `json:"token"`
	TickRate         int    `json:"tick_rate"`
	Password         string `json:"password"`
}

type Admin struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	KleiId string `json:"klei_id"`
}

type LuaValue struct {
	Type   int    `json:"type"`
	String string `json:"string"`
}
