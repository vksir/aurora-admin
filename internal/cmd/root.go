package cmd

import (
	"aurora-admin/internal/app"
	"context"
	"github.com/spf13/cobra"
	"github.com/vksir/vkiss-lib/pkg/cfg"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/thirdpkg/steamcmd"
	"os"
	"os/signal"
	"syscall"
)

var (
	cfgListen = cfg.NewFlag[string]("listen", "listen", "0.0.0.0:5800")
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aurora",
		Short: "aurora admin",
		Run: func(cmd *cobra.Command, args []string) {
			serve()
		},
	}

	steamcmd.CfgSteamcmdPath.Bind(root)
	return root
}

func serve() {
	log.WarnF("aurora admin serve...")
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.DefaultLogger()
	go func() {
		app.Run(ctx, cfgListen.Get(), logger)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	<-sigChan
	cancel()
	log.InfoF("exited")
}
