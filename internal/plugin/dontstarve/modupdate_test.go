package dontstarve

import (
	"testing"

	"aurora-admin/internal/store"
	"aurora-admin/pkg/steam"
	"github.com/stretchr/testify/assert"
)

func detail(id string, timeUpdated int) steam.Publishedfiledetail {
	return steam.Publishedfiledetail{
		Publishedfileid: id,
		Result:          1,
		Title:           "mod-" + id,
		TimeUpdated:     timeUpdated,
	}
}

func TestApplyModInfo(t *testing.T) {
	mod := &store.Mod{WorkshopID: "123", Name: "old", TimeUpdated: 100}
	d := detail("123", 200)
	d.Title = "new name"
	d.PreviewUrl = "http://example.com/img.png"
	d.TimeCreated = 50

	applyModInfo(mod, d)

	assert.Equal(t, "new name", mod.Name)
	assert.Equal(t, "http://example.com/img.png", mod.Image)
	assert.Equal(t, int64(50), mod.TimeCreated)
	// TimeUpdated 不由 applyModInfo 写入，由更新成功后回写
	assert.Equal(t, int64(100), mod.TimeUpdated)
}
