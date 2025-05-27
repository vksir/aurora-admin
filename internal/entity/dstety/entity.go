package dstety

type Status struct {
	Status int `json:"status"`
}

type Config struct {
	EnabledArchiveId string `json:"enabled_archive_id"`
	Token            string `json:"token"`
}
