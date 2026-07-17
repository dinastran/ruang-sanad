package templates

import "strings"

// FooterLink is a single (label, href) pair for footer columns.
type FooterLink struct {
	FLabel string
	FURL   string
}

// FooterLinks is a list of footer links.
type FooterLinks []FooterLink

// toneClasses returns the Tailwind class string for a feature card icon background.
// Allowed values: "brand" (cyan), "secondary" (violet), "success" (emerald).
// Unknown values fall back to the brand tone to keep the grid visually consistent.
func toneClasses(tone string) string {
	switch strings.ToLower(tone) {
	case "brand":
		return "bg-brand-400/10 text-brand-300"
	case "secondary":
		return "bg-secondary-500/10 text-secondary-300"
	case "success":
		return "bg-success/10 text-success"
	default:
		return "bg-brand-400/10 text-brand-300"
	}
}
