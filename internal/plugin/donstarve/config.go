package donstarve

import (
	"aurora-admin/internal/entity/dstety"
	"context"
	"github.com/vksir/vkiss-lib/pkg/cache"
)

var (
	cacheEnabledArchiveId = cache.NewKey[string]("dst.enabled_archive_id")
	cacheToken            = cache.NewKey[string]("dst.token")
)

func (d *Plugin) GetConfig(ctx context.Context) dstety.Config {
	return dstety.Config{
		EnabledArchiveId: cacheEnabledArchiveId.Get(),
		Token:            cacheToken.Get(),
	}
}

func (d *Plugin) UpdateConfig(ctx context.Context, c dstety.Config) dstety.Config {
	if c.EnabledArchiveId != cacheEnabledArchiveId.Get() {
		cacheEnabledArchiveId.Set(c.EnabledArchiveId)
	}
	if c.Token != cacheToken.Get() {
		cacheToken.Set(c.Token)
	}
	return d.GetConfig(ctx)
}
