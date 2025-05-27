package github

import (
	"aurora-admin/pkg/util"
	"encoding/json"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"net/http"
)

func GetLatestRelease(author, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", author, repo)

	resp, err := util.ProxyClient().R().Get(url)
	if err != nil {
		return "", errutil.Wrap(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("bad request: status_code=%d, body=%s",
			resp.StatusCode(), string(resp.Body()))
	}
	var v LatestReleaseResp
	err = json.Unmarshal(resp.Body(), &v)
	if err != nil {
		return "", errutil.Wrap(err)
	}
	return v.TagName, nil
}

func DownLoadRelease(tag, downloadFile, outputPath string) error {
	url := fmt.Sprintf("https://github.com/tModLoader/tModLoader/releases/download/%s/%s",
		tag, downloadFile)
	_, err := util.ProxyClient().R().SetOutput(outputPath).Get(url)
	return err
}
