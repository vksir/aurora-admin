package dstety

import (
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/log"
	lua "github.com/yuin/gopher-lua"
	"strconv"
)

type Status struct {
	Status int `json:"status"`
}

type Config struct {
	EnabledArchiveId string `json:"enabled_archive_id"`
	Token            string `json:"token"`
	TickRate         int    `json:"tick_rate"`
	Password         string `json:"password"`
}

type Admin struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	KleiId string `json:"klei_id"`
}

type LuaValue struct {
	Type   int    `json:"type"`
	String string `json:"string"`
}

func NewLuaValue(lv lua.LValue) LuaValue {
	return LuaValue{Type: int(lv.Type()), String: lv.String()}
}

func (v LuaValue) StringLua() string {
	switch lua.LValueType(v.Type) {
	case lua.LTString:
		return fmt.Sprintf("\"%s\"", v.String)
	case lua.LTBool, lua.LTNumber:
		return v.String
	default:
		log.Error("unsupported lua type",
			"type", lua.LValueType(v.Type), "string", v.String)
		return "\"\""
	}
}

func (v LuaValue) Value() (any, error) {
	switch lua.LValueType(v.Type) {
	case lua.LTString:
		return v.String, nil
	case lua.LTBool:
		vv, err := strconv.ParseBool(v.String)
		if err != nil {
			return nil, fmt.Errorf("ParseBool failed: v=%s, e=%s", v.String, err)
		}
		return vv, nil
	case lua.LTNumber:
		vv, err := strconv.ParseFloat(v.String, 64)
		if err != nil {
			return nil, fmt.Errorf("ParseFloat failed: v=%s, e=%s", v.String, err)
		}
		return vv, nil
	default:
		return nil, fmt.Errorf("unsupported lua type: %s", lua.LValueType(v.Type))
	}
}
