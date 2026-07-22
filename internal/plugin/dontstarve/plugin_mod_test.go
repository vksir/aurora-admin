package dontstarve

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"maps"
	"slices"
	"testing"
)

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

func TestParseAndGenModOverrides(t *testing.T) {
	workshopID2Config, err := parseModLuaByRegex(mockModOverridesLua)
	assert.Nil(t, err)
	res := genModOverrideContent(slices.Collect(maps.Values(workshopID2Config)))
	fmt.Println(res)
}
