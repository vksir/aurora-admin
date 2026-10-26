package dontstarve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchServerCmdline(t *testing.T) {
	const clusterID = "29065466-b5d7-47bd-a224-06f18a36d12d"

	tests := []struct {
		name      string
		args      []string
		clusterID string
		expected  bool
	}{
		{
			name: "Master 分片",
			args: []string{
				"/root/aurora-admin/service/dontstarve/bin64/dontstarve_dedicated_server_nullrenderer_x64",
				"-console", "-cluster", clusterID, "-shard", "Master",
			},
			clusterID: clusterID,
			expected:  true,
		},
		{
			name: "Caves 分片",
			args: []string{
				"/root/aurora-admin/service/dontstarve/bin64/dontstarve_dedicated_server_nullrenderer_x64",
				"-console", "-cluster", clusterID, "-shard", "Caves",
			},
			clusterID: clusterID,
			expected:  true,
		},
		{
			name: "其他集群不匹配",
			args: []string{
				"/root/aurora-admin/service/dontstarve/bin64/dontstarve_dedicated_server_nullrenderer_x64",
				"-console", "-cluster", "another-cluster", "-shard", "Master",
			},
			clusterID: clusterID,
			expected:  false,
		},
		{
			name:      "管理面板进程不匹配",
			args:      []string{"/root/aurora-admin/aurora-admin", "serve", "-c", "/root/aurora-admin/config.toml"},
			clusterID: clusterID,
			expected:  false,
		},
		{
			name: "clusterID 为空时只匹配二进制",
			args: []string{
				"/root/aurora-admin/service/dontstarve/bin64/dontstarve_dedicated_server_nullrenderer_x64",
				"-console",
			},
			clusterID: "",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchServerCmdline(tt.args, tt.clusterID))
		})
	}
}

func TestMatchShardCmdline(t *testing.T) {
	const clusterID = "29065466-b5d7-47bd-a224-06f18a36d12d"
	base := []string{
		"/root/aurora-admin/service/dontstarve/bin64/dontstarve_dedicated_server_nullrenderer_x64",
		"-console", "-cluster", clusterID, "-shard",
	}

	tests := []struct {
		name      string
		args      []string
		clusterID string
		shard     string
		expected  bool
	}{
		{name: "Master 匹配", args: append(append([]string{}, base...), "Master"), clusterID: clusterID, shard: "Master", expected: true},
		{name: "Caves 匹配", args: append(append([]string{}, base...), "Caves"), clusterID: clusterID, shard: "Caves", expected: true},
		{name: "分片不匹配", args: append(append([]string{}, base...), "Caves"), clusterID: clusterID, shard: "Master", expected: false},
		{name: "集群不匹配", args: append(append([]string{}, base...), "Master"), clusterID: "other", shard: "Master", expected: false},
		{name: "非服务端进程", args: []string{"/root/aurora-admin/aurora-admin", "serve"}, clusterID: clusterID, shard: "Master", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchShardCmdline(tt.args, tt.clusterID, tt.shard))
		})
	}
}
