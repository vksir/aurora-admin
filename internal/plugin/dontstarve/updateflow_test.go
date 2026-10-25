package dontstarve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPlayerIDs(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "空输出（无人）",
			output:   "",
			expected: []string{},
		},
		{
			name:     "单个玩家",
			output:   "[00:01:00]: [1] Wilson (KU_aBc123)",
			expected: []string{"KU_aBc123"},
		},
		{
			name: "多个玩家去重",
			output: `[00:01:00]: [1] Wilson (KU_aBc123)
[00:01:00]: [2] Willow (KU_def456)
[00:01:00]: [3] Wilson (KU_aBc123)`,
			expected: []string{"KU_aBc123", "KU_def456"},
		},
		{
			name: "跨 shard 混合输出（Master 与 Caves 各一份）",
			output: `[00:01:00]: [1] WX-78 (KU_111aaa)
[00:02:00]: [1] Wickerbottom (KU_222bbb)
[00:02:00]: [2] WX-78 (KU_111aaa)`,
			expected: []string{"KU_111aaa", "KU_222bbb"},
		},
		{
			name:     "命令回显无 userid",
			output:   "c_listallplayers()",
			expected: []string{},
		},
		{
			name:     "无关输出（公告等）不算玩家",
			output:   "[00:01:00]: [Announcement] server updating soon",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractPlayerIDs(tt.output))
		})
	}
}
