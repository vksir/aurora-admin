package svcety

var (
	StatusUninstalled  = "Uninstalled"
	StatusInstalling   = "Installing"
	StatusUninstalling = "Uninstalling"

	StatusInactive   = "Inactive"
	StatusActive     = "Active"
	StatusStarting   = "Starting"
	StatusStopping   = "Stopping"
	StatusRestarting = "Restarting"
	StatusUpdating   = "Updating"

	ActionStart     = "start"
	ActionStop      = "stop"
	ActionRestart   = "restart"
	ActionInstall   = "install"
	ActionUninstall = "uninstall"
	ActionUpgrade   = "upgrade"
)

type Status struct {
	Status string `json:"status"`
}

type Control struct {
	Action string `json:"action" validate:"oneof=start stop restart install uninstall upgrade"`
}
