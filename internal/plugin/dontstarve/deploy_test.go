package dontstarve

import (
	"aurora-admin/ent"
	"github.com/stretchr/testify/assert"
	"github.com/vksir/vkiss-lib/pkg/template"
	"testing"
)

func TestDeployAdmin(t *testing.T) {
	d := &deployAdmin{Admin: []*ent.DontStarveAdmin{
		{
			KleiID: "id1",
		},
		{
			KleiID: "id2",
		},
	}}
	res, err := template.ExecuteString(d)
	assert.Nil(t, err)
	assert.Equal(t, `id1
id2
`, res)
}

func TestDeployModSetup(t *testing.T) {
	d := &deployModSetup{Mods: []*ent.DontStarveMod{
		{
			WorkshopID: "id1",
		},
		{
			WorkshopID: "id2",
		},
	}}
	res, err := template.ExecuteString(d)
	assert.Nil(t, err)
	assert.Equal(t, `ServerModSetup("id1")
ServerModSetup("id2")
`, res)
}

func TestDeployToken(t *testing.T) {
	d := &deployToken{Token: "token1"}
	res, err := template.ExecuteString(d)
	assert.Nil(t, err)
	assert.Equal(t, `token1`, res)
}
