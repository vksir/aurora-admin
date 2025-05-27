package donstarve

import (
	"fmt"
	"testing"
)

func TestParseModLua(t *testing.T) {
	content := `return {
    ["workshop-1185229307"]={
      configuration_options={
        CAMERA=true,
        CAPTURE=false,
        DAMAGE_NUMBERS=true,
        DAMAGE_RESISTANCE=true,
        FRAME_PHASES=true,
        GLOBAL=false,
        GLOBAL_NUMBERS=false,
        HEADER_CLIENT=false,
        HEADER_SERVER=false,
        HORIZONTAL_OFFSET=0,
        TAG="EPIC",
        TRANSLATOR=false,
        WETNESS_METER=false 
      },
      enabled=false 
    },
    ["workshop-661253977"]={
      configuration_options={ amudiao=true, baodiao=1, kong=0, nillots=0, rendiao=2, zbdiao=true },
      enabled=true 
    } 
  }
`
	res, err := parseModLua(content)
	fmt.Println(err)
	for i, v := range res {
		fmt.Printf("Index %d:\n%+v\n\n", i, v)
	}
}
