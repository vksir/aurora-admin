package convutil

import (
	"encoding/json"
	"fmt"
	"aurora-admin/pkg/util/errutil"
	"strconv"
)

func String(a any) string {
	switch a.(type) {
	case int:
		return strconv.Itoa(a.(int))
	case bool:
		if a.(bool) {
			return "true"
		} else {
			return "false"
		}
	default:
		return fmt.Sprintf("%v", a)
	}
}

func Json(a any) ([]byte, error) {
	return json.MarshalIndent(a, "", "    ")
}

func MustJsonString(a any) string {
	content, err := Json(a)
	errutil.Fatal(err)
	return string(content)
}
