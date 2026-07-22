package util

import (
	"github.com/go-resty/resty/v2"
)

func ProxyClient(proxy string) *resty.Client {
	c := resty.New()
	if proxy != "" {
		c.SetProxy(proxy)
	}
	return c
}

func UpdateIfChanged[T comparable](newValue T, oldValue *T) bool {
	var zero T
	if newValue != zero && newValue != *oldValue {
		*oldValue = newValue
		return true
	}
	return false
}
