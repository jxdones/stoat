package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme defines a set of colors for the UI.
type Theme struct {
	Border        color.Color
	BorderFocused color.Color

	DividerBorder     color.Color
	OverlayBorder color.Color
	OverlayFooter color.Color

	TextMuted   color.Color
	TextPrimary color.Color
	TextAccent  color.Color
	TextWarning color.Color
	TextError   color.Color
	TextHeader  color.Color

	TableHeader color.Color

	TabsPrefix     color.Color
	TabsText       color.Color
	TabsActiveText color.Color
	TabsActiveBg   color.Color

	SidebarTitle    color.Color
	SidebarTitleHot color.Color
}

// Current is the process-wide active theme. Components read it for styling.
// SetNamedTheme updates it when the user changes theme.
var Current = DefaultTheme()

// Default theme. Specifically chosen to match the Stoat logo colors.
func DefaultTheme() Theme {
	return Theme{
		Border:            lipgloss.Color("131"),
		BorderFocused:     lipgloss.Color("98"),
		DividerBorder:     lipgloss.Color("131"),
		OverlayBorder:     lipgloss.Color("131"),
		OverlayFooter:     lipgloss.Color("173"),
		TextMuted:         lipgloss.Color("137"),
		TextPrimary:       lipgloss.Color("180"),
		TextAccent:        lipgloss.Color("141"),
		TextWarning:       lipgloss.Color("179"),
		TextError:         lipgloss.Color("124"),
		TextHeader:        lipgloss.Color("223"),
		TableHeader:       lipgloss.Color("141"),
		TabsPrefix:        lipgloss.Color("138"),
		TabsText:          lipgloss.Color("180"),
		TabsActiveText:    lipgloss.Color("230"),
		TabsActiveBg:      lipgloss.Color("141"),
		SidebarTitle:      lipgloss.Color("223"),
		SidebarTitleHot:   lipgloss.Color("141"),
	}
}

// Dracula theme.
func DraculaTheme() Theme {
	return Theme{
		Border:            lipgloss.Color("60"),
		BorderFocused:     lipgloss.Color("99"),
		DividerBorder:     lipgloss.Color("60"),
		OverlayBorder:     lipgloss.Color("99"),
		OverlayFooter:     lipgloss.Color("145"),
		TextMuted:         lipgloss.Color("145"),
		TextPrimary:       lipgloss.Color("110"),
		TextAccent:        lipgloss.Color("117"),
		TextWarning:       lipgloss.Color("215"),
		TextError:         lipgloss.Color("203"),
		TextHeader:        lipgloss.Color("231"),
		TableHeader:       lipgloss.Color("99"),
		TabsPrefix:        lipgloss.Color("110"),
		TabsText:          lipgloss.Color("189"),
		TabsActiveText:    lipgloss.Color("231"),
		TabsActiveBg:      lipgloss.Color("99"),
		SidebarTitle:      lipgloss.Color("225"),
		SidebarTitleHot:   lipgloss.Color("117"),
	}
}

// Solarized theme.
func SolarizedTheme() Theme {
	return Theme{
		Border:            lipgloss.Color("240"),
		BorderFocused:     lipgloss.Color("37"),
		DividerBorder:     lipgloss.Color("240"),
		OverlayBorder:     lipgloss.Color("37"),
		OverlayFooter:     lipgloss.Color("244"),
		TextMuted:         lipgloss.Color("244"),
		TextPrimary:       lipgloss.Color("145"),
		TextAccent:        lipgloss.Color("37"),
		TextWarning:       lipgloss.Color("166"),
		TextError:         lipgloss.Color("160"),
		TextHeader:        lipgloss.Color("230"),
		TableHeader:       lipgloss.Color("37"),
		TabsPrefix:        lipgloss.Color("136"),
		TabsText:          lipgloss.Color("145"),
		TabsActiveText:    lipgloss.Color("230"),
		TabsActiveBg:      lipgloss.Color("37"),
		SidebarTitle:      lipgloss.Color("187"),
		SidebarTitleHot:   lipgloss.Color("37"),
	}
}

// SetNamedTheme sets the process-wide theme by name and returns that theme.
// The second return is false if the name is unknown; then the theme is unchanged and
// the returned theme is the current one (caller can ignore it or keep using Current).
func SetNamedTheme(name string) (Theme, bool) {
	var th Theme
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default":
		th = DefaultTheme()
	case "dracula":
		th = DraculaTheme()
	case "solarized":
		th = SolarizedTheme()
	default:
		return Current, false
	}
	Current = th
	return th, true
}
