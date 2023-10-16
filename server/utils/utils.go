package utils

// DeduplicateStringSlice removes duplicate entries from a slice of strings.
func DeduplicateStringSlice(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if _, value := keys[item]; !value {
			result = append(result, item)
			keys[item] = true
		}
	}

	return result
}

// Max returns the larger of two int64 values.
func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of two int64 values.
func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
