package slice

// MergeInPlace 原地合并（不创建新切片，覆盖原切片）
func MergeInPlace(slice1, slice2 []uint32) []uint32 {
	// 计算需要的总容量
	totalLen := len(slice1) + len(slice2)

	// 如果slice1容量不足，创建一个新的足够大的切片
	if cap(slice1) < totalLen {
		newSlice := make([]uint32, len(slice1), totalLen)
		copy(newSlice, slice1)
		slice1 = newSlice
	}

	// 扩展slice1的长度，并复制slice2的元素
	slice1 = slice1[:totalLen]
	copy(slice1[len(slice1)-len(slice2):], slice2)

	return slice1
}

// MergeAndDeduplicateOrdered 有序去重合并（不允许重复元素，保持原顺序）
func MergeAndDeduplicateOrdered(slice1, slice2 []uint32) []uint32 {
	seen := make(map[uint32]struct{})
	result := make([]uint32, 0, len(slice1)+len(slice2))

	// 先添加slice1的元素（保持顺序）
	for _, v := range slice1 {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	// 再添加slice2的元素（跳过已存在的）
	for _, v := range slice2 {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

// MergeAndDeduplicate 去重合并（不允许重复元素，无序）
func MergeAndDeduplicate(slice1, slice2 []uint32) []uint32 {
	set := make(map[uint32]struct{})
	for _, v := range slice1 {
		set[v] = struct{}{}
	}
	for _, v := range slice2 {
		set[v] = struct{}{}
	}

	result := make([]uint32, 0, len(set))
	for v := range set {
		result = append(result, v)
	}
	return result
}

// Unique 对切片进行去重，保持元素原有顺序。
// 泛型类型 T 需要是 comparable，以便用作 map 的键。
func Unique[T comparable](s []T) []T {
	if len(s) == 0 {
		return s
	}
	seen := make(map[T]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
