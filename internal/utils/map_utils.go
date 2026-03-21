package utils

// MergeMapWithPrefix copies all entries from add into existing, prepending prefix to each key.
func MergeMapWithPrefix(prefix string, existing, add map[string]string) {
	for k, v := range add {
		existing[prefix+k] = v
	}
}

// MergeMap copies all entries from add into existing, overwriting duplicates.
func MergeMap(existing, add map[string]string) {
	for k, v := range add {
		existing[k] = v
	}
}
