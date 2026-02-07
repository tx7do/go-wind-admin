package utils

func FilterBlacklist(data []string, blacklist []string) []string {
	bm := make(map[string]struct{}, len(blacklist))
	for _, s := range blacklist {
		bm[s] = struct{}{}
	}

	n := 0
	for _, x := range data {
		if _, found := bm[x]; !found {
			data[n] = x
			n++
		}
	}
	return data[:n]
}
