package registry

import (
	"context"
	"aurora-admin/pkg/log"
	"time"
)

var gCronJobCancel = make(map[string]context.CancelFunc)

type CronJob func(ctx context.Context) error

func RegisterCronJob(name string, interval time.Duration, job CronJob) {
	ctx, cancel := context.WithCancel(context.Background())
	gCronJobCancel[name] = cancel

	notifyChan := make(chan struct{}, 1)
	Subscribe(getCronJobTopic(name), name, func(ctx context.Context, msgAny any) error {
		select {
		case <-ctx.Done():
		case <-notifyChan:
		}
		return nil
	})

	ctx = log.AppendCtx(ctx, "cron_job", name)
	log.InfoC(ctx, "register cron job")

	go func() {
		timer := time.NewTimer(interval)
		for {
			select {
			case <-ctx.Done():
				log.InfoC(ctx, "exit cron job")
				return
			case <-timer.C:
				log.DebugC(ctx, "begin exec cron job by timer")
			case <-notifyChan:
				log.DebugC(ctx, "begin exec cron job by notify")
			}

			err := job(ctx)
			if err != nil {
				log.ErrorC(ctx, "exec cron job failed", "err", err)
			}
			timer.Reset(interval)
		}
	}()
}

func TriggerCronJob(ctx context.Context, name string) {
	Notify(ctx, getCronJobTopic(name), nil)
}

func getCronJobTopic(name string) string {
	return "cron_job_" + name
}
