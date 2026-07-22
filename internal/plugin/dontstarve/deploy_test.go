package dontstarve

import (
	"strings"
	"testing"
	"text/template"

	"aurora-admin/internal/store"
	"github.com/stretchr/testify/assert"
)

func executeString(tmpl textTemplate) (string, error) {
	t, err := template.New("").Parse(tmpl.Template())
	if err != nil {
		return "", err
	}
	buf := &strings.Builder{}
	err = t.Execute(buf, tmpl)
	return buf.String(), err
}

func TestDeployAdmin(t *testing.T) {
	d := &deployAdmin{Admin: []*store.Admin{
		{
			KleiID: "id1",
		},
		{
			KleiID: "id2",
		},
	}}
	res, err := executeString(d)
	assert.Nil(t, err)
	assert.Equal(t, `id1
id2
`, res)
}

func TestDeployModSetup(t *testing.T) {
	d := &deployModSetup{Mods: []*store.Mod{
		{
			WorkshopID: "id1",
		},
		{
			WorkshopID: "id2",
		},
	}}
	res, err := executeString(d)
	assert.Nil(t, err)
	assert.Equal(t, `ServerModSetup("id1")
ServerModSetup("id2")
`, res)
}

func TestDeployToken(t *testing.T) {
	d := &deployToken{Token: "token1"}
	res, err := executeString(d)
	assert.Nil(t, err)
	assert.Equal(t, `token1`, res)
}
