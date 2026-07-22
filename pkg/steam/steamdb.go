package steam

import (
	"github.com/mmcdole/gofeed"
	"aurora-admin/pkg/util/errutil"
	"net/url"
)

func GetPatchNotesRSS(appId string) (*gofeed.Feed, error) {
	u, _ := url.Parse("https://steamdb.info/api/PatchnotesRSS/")
	q := u.Query()
	q.Set("appid", appId)
	u.RawQuery = q.Encode()

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(u.String())
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return feed, nil
}
