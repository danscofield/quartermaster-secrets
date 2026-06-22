package auth

// Intersects returns true if any element appears in both slices.
func Intersects(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// CanRead returns true if the caller holds an owner or retriever billet.
func CanRead(callerBillets, owners, retrievers []string) bool {
	return Intersects(callerBillets, owners) || Intersects(callerBillets, retrievers)
}

// CanUpdate returns true if the caller holds an owner billet.
func CanUpdate(callerBillets, owners []string) bool {
	return Intersects(callerBillets, owners)
}

// ValidateOwnerIntersection requires at least one owner billet to match the caller.
func ValidateOwnerIntersection(callerBillets, owners []string) bool {
	return Intersects(callerBillets, owners)
}
