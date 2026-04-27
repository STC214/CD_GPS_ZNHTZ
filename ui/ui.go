package ui

// Theme is the shared frontend surface for small visual adjustments.
type Theme struct {
	Name string
}

// DefaultTheme returns the shared theme used by the frontend.
func DefaultTheme() Theme {
	return Theme{Name: "default"}
}
