package comety

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
	Code    int    `json:"code"`
}

const (
	ActionStart     = "start"
	ActionStop      = "stop"
	ActionRestart   = "restart"
	ActionInstall   = "install"
	ActionUninstall = "uninstall"
	ActionUpdate    = "update"
)

type Status struct {
	Status string `json:"status"`
}

type Control struct {
	Action string `json:"action" validate:"oneof=start stop restart install uninstall update"`
}
