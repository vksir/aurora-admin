package github

import (
	"aurora-admin/internal/err"
	"aurora-admin/pkg/util"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetLatestRelease(author, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", author, repo)

	resp, err := util.ProxyClient().R().Get(url)
	if err != nil {
		return "", err.NewErr(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", err.NewErr(resp)
	}
	var v LatestReleaseResp
	if err := json.Unmarshal(resp.Body(), &v); err != nil {
		return "", err
	}

	return v.TagName, nil
}

func DownLoadRelease(tag, downloadFile, outputPath string) error {
	url := fmt.Sprintf("https://github.com/tModLoader/tModLoader/releases/download/%s/%s", tag, downloadFile)

	_, err := util.ProxyClient().R().SetOutput(outputPath).Get(url)
	return err
}
