package model

import (
	"errors"
	"regexp"
	"slices"
)

// AppearanceProfile is the portable subset of the X-Panel theme settings.
// The JSON contract is shared with the X-Panel node and Dashboard UI.
type AppearanceProfile struct {
	SchemaVersion int    `json:"schemaVersion"`
	ThemeID       string `json:"themeId"`
	Mode          string `json:"mode"`
	Density       string `json:"density,omitempty"`
	Font          string `json:"font,omitempty"`
	Radius        string `json:"radius,omitempty"`
	SidebarWidth  string `json:"sidebarWidth,omitempty"`
	Accent        string `json:"accent,omitempty"`
}

var appearanceAccent = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (p AppearanceProfile) Validate() error {
	if p.SchemaVersion != 1 ||
		!slices.Contains([]string{"atelier", "lumen", "ink", "harbor", "quartz"}, p.ThemeID) ||
		!slices.Contains([]string{"auto", "light", "dark"}, p.Mode) ||
		!slices.Contains([]string{"", "compact", "default", "comfortable"}, p.Density) ||
		!slices.Contains([]string{"", "system", "inter", "noto", "lxgw"}, p.Font) ||
		!slices.Contains([]string{"", "sharp", "default", "rounded"}, p.Radius) ||
		!slices.Contains([]string{"", "narrow", "default", "wide"}, p.SidebarWidth) ||
		(p.Accent != "" && !appearanceAccent.MatchString(p.Accent)) {
		return errors.New("invalid appearance profile")
	}
	return nil
}
