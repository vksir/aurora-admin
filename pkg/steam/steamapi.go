package steam

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"

	"aurora-admin/pkg/util/errutil"
)

type CmdAppInfoResponse struct {
	Data struct {
		Field1 struct {
			ChangeNumber int    `json:"_change_number"`
			MissingToken bool   `json:"_missing_token"`
			Sha          string `json:"_sha"`
			Size         int    `json:"_size"`
			Appid        string `json:"appid"`
			Common       struct {
				Associations struct {
				} `json:"associations"`
				Clienticns string `json:"clienticns"`
				Clienticon string `json:"clienticon"`
				Clienttga  string `json:"clienttga"`
				Eulas      struct {
					Field1 struct {
						Id   string `json:"id"`
						Name string `json:"name"`
						Url  string `json:"url"`
					} `json:"0"`
				} `json:"eulas"`
				Freetodownload  string `json:"freetodownload"`
				Gameid          string `json:"gameid"`
				Icon            string `json:"icon"`
				Linuxclienticon string `json:"linuxclienticon"`
				Logo            string `json:"logo"`
				LogoSmall       string `json:"logo_small"`
				Name            string `json:"name"`
				Osarch          string `json:"osarch"`
				Osextended      string `json:"osextended"`
				Oslist          string `json:"oslist"`
				Releasestate    string `json:"releasestate"`
				SectionType     string `json:"section_type"`
				Type            string `json:"type"`
			} `json:"common"`
			Config struct {
				Contenttype string `json:"contenttype"`
				Installdir  string `json:"installdir"`
				Launch      struct {
					Field1 struct {
						Config struct {
							Osarch string `json:"osarch"`
							Oslist string `json:"oslist"`
						} `json:"config"`
						Description    string `json:"description"`
						DescriptionLoc struct {
							English string `json:"english"`
						} `json:"description_loc"`
						Executable string `json:"executable"`
						Type       string `json:"type"`
						Workingdir string `json:"workingdir"`
					} `json:"0"`
					Field2 struct {
						Config struct {
							Osarch string `json:"osarch"`
							Oslist string `json:"oslist"`
						} `json:"config"`
						Description    string `json:"description"`
						DescriptionLoc struct {
							English string `json:"english"`
						} `json:"description_loc"`
						Executable string `json:"executable"`
						Type       string `json:"type"`
						Workingdir string `json:"workingdir"`
					} `json:"1"`
					Field3 struct {
						Config struct {
							Oslist string `json:"oslist"`
						} `json:"config"`
						Description    string `json:"description"`
						DescriptionLoc struct {
							English string `json:"english"`
						} `json:"description_loc"`
						Executable string `json:"executable"`
						Type       string `json:"type"`
					} `json:"2"`
					Field4 struct {
						Config struct {
							Osarch string `json:"osarch"`
							Oslist string `json:"oslist"`
						} `json:"config"`
						Description    string `json:"description"`
						DescriptionLoc struct {
							English string `json:"english"`
						} `json:"description_loc"`
						Executable string `json:"executable"`
						Type       string `json:"type"`
						Workingdir string `json:"workingdir"`
					} `json:"3"`
					Field5 struct {
						Config struct {
							Osarch string `json:"osarch"`
							Oslist string `json:"oslist"`
						} `json:"config"`
						Description    string `json:"description"`
						DescriptionLoc struct {
							English string `json:"english"`
						} `json:"description_loc"`
						Executable string `json:"executable"`
						Type       string `json:"type"`
						Workingdir string `json:"workingdir"`
					} `json:"4"`
				} `json:"launch"`
			} `json:"config"`
			Depots struct {
				Field1 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Depotfromapp string `json:"depotfromapp"`
					Manifests    struct {
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
					} `json:"manifests"`
				} `json:"1004"`
				Field2 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Depotfromapp string `json:"depotfromapp"`
					Manifests    struct {
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
					} `json:"manifests"`
				} `json:"1005"`
				Field3 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Depotfromapp string `json:"depotfromapp"`
					Manifests    struct {
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
					} `json:"manifests"`
				} `json:"1006"`
				Field4 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Depotfromapp  string `json:"depotfromapp"`
					Sharedinstall string `json:"sharedinstall"`
				} `json:"228982"`
				Field5 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Depotfromapp  string `json:"depotfromapp"`
					Sharedinstall string `json:"sharedinstall"`
				} `json:"228988"`
				Field6 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Manifests struct {
						Beforemacoschanges struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"beforemacoschanges"`
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
						Updatebeta struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"updatebeta"`
					} `json:"manifests"`
				} `json:"343051"`
				Field7 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Manifests struct {
						Beforemacoschanges struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"beforemacoschanges"`
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
						Updatebeta struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"updatebeta"`
					} `json:"manifests"`
				} `json:"343052"`
				Field8 struct {
					Config struct {
						Oslist string `json:"oslist"`
					} `json:"config"`
					Manifests struct {
						Beforemacoschanges struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"beforemacoschanges"`
						Public struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"public"`
						Updatebeta struct {
							Download string `json:"download"`
							Gid      string `json:"gid"`
							Size     string `json:"size"`
						} `json:"updatebeta"`
					} `json:"manifests"`
				} `json:"343053"`
				Branches struct {
					Beforemacoschanges struct {
						Buildid          string `json:"buildid"`
						Description      string `json:"description"`
						Timebuildupdated string `json:"timebuildupdated"`
						Timeupdated      string `json:"timeupdated"`
					} `json:"beforemacoschanges"`
					Public struct {
						Buildid          string `json:"buildid"`
						Timebuildupdated string `json:"timebuildupdated"`
						Timeupdated      string `json:"timeupdated"`
					} `json:"public"`
					Updatebeta struct {
						Buildid          string `json:"buildid"`
						Description      string `json:"description"`
						Timebuildupdated string `json:"timebuildupdated"`
						Timeupdated      string `json:"timeupdated"`
					} `json:"updatebeta"`
				} `json:"branches"`
				Privatebranches string `json:"privatebranches"`
			} `json:"depots"`
			Extended struct {
				Gamedir string `json:"gamedir"`
			} `json:"extended"`
		} `json:"343050"`
	} `json:"data"`
	Status string `json:"status"`
}

func GetSteamCmdAppInfo(appID string) (CmdAppInfoResponse, error) {
	var res CmdAppInfoResponse
	url := fmt.Sprintf("https://api.steamcmd.net/v1/info/%s", appID)
	err := request(http.MethodGet, url, map[string]string{}, &res)
	return res, err
}

type Publishedfiledetail struct {
	Publishedfileid       string `json:"publishedfileid"`
	Result                int    `json:"result"`
	Creator               string `json:"creator"`
	CreatorAppId          int    `json:"creator_app_id"`
	ConsumerAppId         int    `json:"consumer_app_id"`
	Filename              string `json:"filename"`
	FileSize              string `json:"file_size"`
	FileUrl               string `json:"file_url"`
	HcontentFile          string `json:"hcontent_file"`
	PreviewUrl            string `json:"preview_url"`
	HcontentPreview       string `json:"hcontent_preview"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	TimeCreated           int    `json:"time_created"`
	TimeUpdated           int    `json:"time_updated"`
	Visibility            int    `json:"visibility"`
	Banned                int    `json:"banned"`
	BanReason             string `json:"ban_reason"`
	Subscriptions         int    `json:"subscriptions"`
	Favorited             int    `json:"favorited"`
	LifetimeSubscriptions int    `json:"lifetime_subscriptions"`
	LifetimeFavorited     int    `json:"lifetime_favorited"`
	Views                 int    `json:"views"`
	Tags                  []struct {
		Tag string `json:"tag"`
	} `json:"tags"`
}

type GetPublishedFileDetailsResponse struct {
	Response struct {
		Result               int                   `json:"result"`
		Resultcount          int                   `json:"resultcount"`
		Publishedfiledetails []Publishedfiledetail `json:"publishedfiledetails"`
	} `json:"response"`
}

func GetPublishedFileDetails(workshopID ...string) (GetPublishedFileDetailsResponse, error) {
	var res GetPublishedFileDetailsResponse
	if len(workshopID) == 0 {
		return res, fmt.Errorf("workshopID is empty")
	}

	url := "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/"
	data := map[string]string{
		"itemcount": strconv.Itoa(len(workshopID)),
	}
	for i, id := range workshopID {
		key := fmt.Sprintf("publishedfileids[%d]", i)
		data[key] = id
	}
	err := request(http.MethodPost, url, data, &res)
	return res, err
}

type GetNewsForAppResponse struct {
	Appnews struct {
		Appid     int `json:"appid"`
		Newsitems []struct {
			Gid           string   `json:"gid"`
			Title         string   `json:"title"`
			Url           string   `json:"url"`
			IsExternalUrl bool     `json:"is_external_url"`
			Author        string   `json:"author"`
			Contents      string   `json:"contents"`
			Feedlabel     string   `json:"feedlabel"`
			Date          int      `json:"date"`
			Feedname      string   `json:"feedname"`
			FeedType      int      `json:"feed_type"`
			Appid         int      `json:"appid"`
			Tags          []string `json:"tags"`
		} `json:"newsitems"`
		Count int `json:"count"`
	} `json:"appnews"`
}

func GetNewsForApp(appId string) (GetNewsForAppResponse, error) {
	url := "https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/"
	data := map[string]string{
		"appid": appId,
	}
	var res GetNewsForAppResponse
	err := request(http.MethodGet, url, data, &res)
	return res, err
}

// httpClient 为包级共享客户端，复用连接池，
// 避免每次请求重建 TCP/TLS 连接（每次握手约多耗 100ms+）。
var httpClient = resty.New().
	SetTimeout(10 * time.Second).
	SetTransport(&http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	})

func request(method, url string, data map[string]string, a any) error {
	client := httpClient.R()
	if method == http.MethodGet {
		client.SetQueryParams(data)
	} else {
		client.SetHeader("Content-Type", "application/x-www-form-urlencoded").
			SetFormData(data)
	}
	resp, err := client.Execute(method, url)
	if err != nil {
		return errutil.Wrap(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return errutil.WrapF("request failed: code=%d, body=%s", resp.StatusCode(), resp.Body())
	}
	err = json.Unmarshal(resp.Body(), a)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}
