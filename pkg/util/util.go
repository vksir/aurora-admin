package util

import (
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
)

func ProxyClient() *resty.Client {
	c := resty.New()
	if proxy := viper.GetString("ns.proxy"); proxy != "" {
		c.SetProxy(proxy)
	}
	return c
}
