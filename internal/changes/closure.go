package changes

import "slices"

// Expand returns the sorted names reachable from start by following next,
// including start itself.
func Expand(start []string, next func(string) []string) []string {
	seen := make(map[string]bool, len(start))
	queue := slices.Clone(start)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		queue = append(queue, next(name)...)
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}
