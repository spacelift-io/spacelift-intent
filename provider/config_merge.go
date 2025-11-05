package provider

// mergeMaps recursively merges source into base, using base as the foundation
// and overwriting with any fields that appear in source. This ensures that fields present
// in base but missing from source are preserved.
func mergeMaps(base, source map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any)
	}
	if source == nil {
		return base
	}

	// Create a copy of base to avoid mutating the original
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}

	// Recursively merge source into result
	for key, sourceValue := range source {
		if sourceValue == nil {
			// Explicit nil in source overwrites
			result[key] = nil
			continue
		}

		baseValue, exists := result[key]
		if !exists {
			// Key doesn't exist in base, just set it
			result[key] = sourceValue
			continue
		}

		// Both values exist, check if they are both maps for recursive merge
		baseMap, baseIsMap := baseValue.(map[string]any)
		sourceMap, sourceIsMap := sourceValue.(map[string]any)

		if baseIsMap && sourceIsMap {
			// Both are maps, recursively merge
			result[key] = mergeMaps(baseMap, sourceMap)
		} else {
			// Not both maps, source overwrites
			result[key] = sourceValue
		}
	}

	return result
}
