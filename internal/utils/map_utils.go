package utils

func MergeMapWithPrefix(prefix string, existing, add map[string]string) {
	for k, v := range add {
		existing[prefix+k] = v
	}
}

func MergeMap(existing, add map[string]string) {
	for k, v := range add {
		existing[k] = v
	}
}
