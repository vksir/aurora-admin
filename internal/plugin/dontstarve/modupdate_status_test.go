package dontstarve

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 前端 ModUpdateStatus 依赖 current_time / latest_time / need_update 三个字段名
func TestModUpdateStatusJSONFields(t *testing.T) {
	raw, err := json.Marshal(ModUpdateStatus{
		WorkshopID:     "2893991859",
		Name:           "CookDiary",
		CurrentUpdated: 100,
		RemoteUpdated:  200,
		NeedUpdate:     true,
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "2893991859", got["workshop_id"])
	assert.Equal(t, "CookDiary", got["name"])
	assert.Equal(t, float64(100), got["current_time"])
	assert.Equal(t, float64(200), got["latest_time"])
	assert.Equal(t, true, got["need_update"])
}
