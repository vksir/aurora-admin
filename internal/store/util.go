package store

// nonNil 保证列表查询在无结果时返回空切片而非 nil，
// 避免 JSON 序列化为 null 导致前端崩溃。
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
