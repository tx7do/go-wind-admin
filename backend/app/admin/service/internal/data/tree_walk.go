package data

// collectTreeIDs 按 parent→children 映射内存收集 rootID 及其全部后代（含自身），
// 代替 UNION ALL 递归 CTE 做子树收集：
//   - 方言全覆盖（go-crud 的 QueryAllChildrenIds 递归 CTE 只有 MySQL/Postgres
//     分支，其余方言静默返回空集——删除会只删根、整条子树变孤儿）；
//   - visited 防护让存量环脏数据（外部导入/直改库）正常收敛，不打转挂死。
func collectTreeIDs[V any](childrenOf map[uint64][]V, nodeID func(V) uint64, rootID uint64) []uint64 {
	ids := make([]uint64, 0, 8)
	visited := map[uint64]struct{}{rootID: {}}
	stack := []uint64{rootID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		ids = append(ids, cur)
		for _, ch := range childrenOf[cur] {
			id := nodeID(ch)
			if _, seen := visited[id]; seen {
				continue
			}
			visited[id] = struct{}{}
			stack = append(stack, id)
		}
	}
	return ids
}
