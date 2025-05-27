package dontstarve

import (
	"aurora-admin/internal/entity/dstety"
	"fmt"
	"github.com/stretchr/testify/assert"
	"maps"
	"slices"
	"testing"
)

const mockModInfoLua = `
name = "mock_name"
author = "mock_author"
version = "mock_version"
description = "mock_description"
configuration_options =
{
    {
        name = "mock_config1",
        label = "mock_label1",
        options =
        {
            {description = "中文", data = 1, hover = "中文"},
            {description = "English", data = 2, hover = "English"},
        },
        default = 1
    },
    {
        name = "配置",
        label = "标签",
        options =
        {
            {description = "开", data = true},
            {description = "关", data = false}
        },
        default = true
    }
}
`

const mockModOverridesLua = `return {
  ["workshop-1392778117"]={
    configuration_options={
      Language="chinese",
      LilyBushSpacing=1,
      ["UI相关"]=false
    },
    enabled=true 
  },
  ["workshop-1991746508"]={ configuration_options={ Language="A", ShowBuff=true }, enabled=true }
}`

func TestFetchModInfoFromLua(t *testing.T) {
	var mod dstety.DontStarveMod
	mod.WorkshopId = "1234561"
	err := fetchModInfoFromLua([]byte(mockModInfoLua), &mod)
	assert.Nil(t, err)

	// 测试有分配到 default option
	for _, c := range mod.Config {
		hasDefault := false
		for _, o := range c.Option {
			if o.Default {
				hasDefault = true
			}
		}
		assert.True(t, hasDefault)
	}
}

func TestParseAndGenModOverrides(t *testing.T) {
	workshopID2Config, err := parseModLuaByRegex(mockModOverridesLua)
	assert.Nil(t, err)
	res := genModOverrideContent(slices.Collect(maps.Values(workshopID2Config)))
	fmt.Println(res)
}
