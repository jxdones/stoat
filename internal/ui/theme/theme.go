package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Theme defines a set of colors for the UI.
type Theme struct {
	Border        color.Color
	BorderFocused color.Color

	DividerBorder color.Color
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

// cc creates a CompleteColor with true color, ANSI256, and ANSI (16-color) fallbacks.
func cc(hex, ansi256, ansi string) color.Color {
	return compat.CompleteColor{
		TrueColor: lipgloss.Color(hex),
		ANSI256:   lipgloss.Color(ansi256),
		ANSI:      lipgloss.Color(ansi),
	}
}

// Default theme. Specifically chosen to match the Stoat logo colors.
func DefaultTheme() Theme {
	return Theme{
		Border:          cc("#af5f5f", "131", "1"),
		BorderFocused:   cc("#875fd7", "98", "5"),
		DividerBorder:   cc("#af5f5f", "131", "1"),
		OverlayBorder:   cc("#af5f5f", "131", "1"),
		OverlayFooter:   cc("#d7875f", "173", "3"),
		TextMuted:       cc("#af875f", "137", "8"),
		TextPrimary:     cc("#d7af87", "180", "7"),
		TextAccent:      cc("#af87ff", "141", "5"),
		TextWarning:     cc("#d7af5f", "179", "3"),
		TextError:       cc("#af0000", "124", "1"),
		TextHeader:      cc("#ffd7af", "223", "15"),
		TableHeader:     cc("#af87ff", "141", "5"),
		TabsPrefix:      cc("#af8787", "138", "8"),
		TabsText:        cc("#d7af87", "180", "7"),
		TabsActiveText:  cc("#ffffd7", "230", "15"),
		TabsActiveBg:    cc("#af87ff", "141", "5"),
		SidebarTitle:    cc("#ffd7af", "223", "15"),
		SidebarTitleHot: cc("#af87ff", "141", "5"),
	}
}

// Dracula theme.
func DraculaTheme() Theme {
	return Theme{
		Border:          cc("#5f5f87", "60", "8"),
		BorderFocused:   cc("#875fff", "99", "5"),
		DividerBorder:   cc("#5f5f87", "60", "8"),
		OverlayBorder:   cc("#875fff", "99", "5"),
		OverlayFooter:   cc("#afafaf", "145", "7"),
		TextMuted:       cc("#afafaf", "145", "8"),
		TextPrimary:     cc("#87afd7", "110", "7"),
		TextAccent:      cc("#87d7ff", "117", "14"),
		TextWarning:     cc("#ffaf5f", "215", "3"),
		TextError:       cc("#ff5f5f", "203", "1"),
		TextHeader:      cc("#ffffff", "231", "15"),
		TableHeader:     cc("#875fff", "99", "5"),
		TabsPrefix:      cc("#87afd7", "110", "7"),
		TabsText:        cc("#d7d7ff", "189", "7"),
		TabsActiveText:  cc("#ffffff", "231", "15"),
		TabsActiveBg:    cc("#875fff", "99", "5"),
		SidebarTitle:    cc("#ffd7ff", "225", "15"),
		SidebarTitleHot: cc("#87d7ff", "117", "14"),
	}
}

// Solarized theme.
func SolarizedTheme() Theme {
	return Theme{
		Border:          cc("#585858", "240", "8"),
		BorderFocused:   cc("#00afaf", "37", "6"),
		DividerBorder:   cc("#585858", "240", "8"),
		OverlayBorder:   cc("#00afaf", "37", "6"),
		OverlayFooter:   cc("#808080", "244", "8"),
		TextMuted:       cc("#808080", "244", "8"),
		TextPrimary:     cc("#afafaf", "145", "7"),
		TextAccent:      cc("#00afaf", "37", "6"),
		TextWarning:     cc("#d75f00", "166", "3"),
		TextError:       cc("#d70000", "160", "1"),
		TextHeader:      cc("#ffffd7", "230", "15"),
		TableHeader:     cc("#00afaf", "37", "6"),
		TabsPrefix:      cc("#af8700", "136", "3"),
		TabsText:        cc("#afafaf", "145", "7"),
		TabsActiveText:  cc("#ffffd7", "230", "15"),
		TabsActiveBg:    cc("#00afaf", "37", "6"),
		SidebarTitle:    cc("#d7d7af", "187", "7"),
		SidebarTitleHot: cc("#00afaf", "37", "6"),
	}
}

// M365Princess theme (oh-my-posh: M365Princess.omp.json).
func PrincessTheme() Theme {
	return Theme{
		Border:          cc("#33658A", "67", "4"),
		BorderFocused:   cc("#9A348E", "127", "5"),
		DividerBorder:   cc("#33658A", "67", "4"),
		OverlayBorder:   cc("#9A348E", "127", "5"),
		OverlayFooter:   cc("#FCA17D", "216", "3"),
		TextMuted:       cc("#86BBD8", "110", "7"),
		TextPrimary:     cc("#FFFFFF", "231", "7"),
		TextAccent:      cc("#DA627D", "168", "5"),
		TextWarning:     cc("#FCA17D", "216", "3"),
		TextError:       cc("#CC3802", "130", "1"),
		TextHeader:      cc("#FFFFFF", "231", "15"),
		TableHeader:     cc("#047E84", "30", "6"),
		TabsPrefix:      cc("#86BBD8", "110", "7"),
		TabsText:        cc("#FFFFFF", "231", "7"),
		TabsActiveText:  cc("#FFFFFF", "231", "15"),
		TabsActiveBg:    cc("#9A348E", "127", "5"),
		SidebarTitle:    cc("#FCA17D", "216", "3"),
		SidebarTitleHot: cc("#DA627D", "168", "5"),
	}
}

// 1_shell theme (oh-my-posh: 1_shell.omp.json).
func OneShellTheme() Theme {
	return Theme{
		Border:          cc("#bc93ff", "139", "13"),
		BorderFocused:   cc("#00c7fc", "45", "14"),
		DividerBorder:   cc("#bc93ff", "139", "13"),
		OverlayBorder:   cc("#00c7fc", "45", "14"),
		OverlayFooter:   cc("#ffbebc", "217", "7"),
		TextMuted:       cc("#ffafd2", "218", "8"),
		TextPrimary:     cc("#FEF5ED", "230", "7"),
		TextAccent:      cc("#ff70a6", "205", "13"),
		TextWarning:     cc("#a9ffb4", "157", "10"),
		TextError:       cc("#ef5350", "203", "1"),
		TextHeader:      cc("#ffffff", "231", "15"),
		TableHeader:     cc("#bc93ff", "183", "13"),
		TabsPrefix:      cc("#00c7fc", "45", "14"),
		TabsText:        cc("#FEF5ED", "230", "7"),
		TabsActiveText:  cc("#ffffff", "231", "15"),
		TabsActiveBg:    cc("#ee79d1", "170", "13"),
		SidebarTitle:    cc("#ffbebc", "217", "7"),
		SidebarTitleHot: cc("#ff70a6", "205", "13"),
	}
}

// blueish theme (oh-my-posh: blueish.omp.json).
func BlueishTheme() Theme {
	return Theme{
		Border:          cc("#546E7A", "66", "8"),
		BorderFocused:   cc("#26C6DA", "44", "14"),
		DividerBorder:   cc("#546E7A", "66", "8"),
		OverlayBorder:   cc("#26C6DA", "44", "14"),
		OverlayFooter:   cc("#a2beef", "153", "7"),
		TextMuted:       cc("#a2c4e0", "153", "8"),
		TextPrimary:     cc("#ffffff", "231", "7"),
		TextAccent:      cc("#14c2dd", "44", "14"),
		TextWarning:     cc("#FFCD58", "221", "3"),
		TextError:       cc("#f1184c", "197", "1"),
		TextHeader:      cc("#ffffff", "231", "15"),
		TableHeader:     cc("#26C6DA", "44", "14"),
		TabsPrefix:      cc("#a2c4e0", "153", "7"),
		TabsText:        cc("#ffffff", "231", "7"),
		TabsActiveText:  cc("#ffffff", "231", "15"),
		TabsActiveBg:    cc("#0476d0", "32", "4"),
		SidebarTitle:    cc("#a2beef", "153", "7"),
		SidebarTitleHot: cc("#26C6DA", "44", "14"),
	}
}

// Rosé Pine theme.
func RosePineTheme() Theme {
	return Theme{
		Border:          cc("#908caa", "103", "8"),
		BorderFocused:   cc("#c4a7e7", "183", "5"),
		DividerBorder:   cc("#6e6a86", "60", "8"),
		OverlayBorder:   cc("#c4a7e7", "183", "5"),
		OverlayFooter:   cc("#f6c177", "222", "3"),
		TextMuted:       cc("#6e6a86", "103", "8"),
		TextPrimary:     cc("#e0def4", "189", "7"),
		TextAccent:      cc("#c4a7e7", "183", "5"),
		TextWarning:     cc("#f6c177", "222", "3"),
		TextError:       cc("#eb6f92", "168", "1"),
		TextHeader:      cc("#ebbcba", "181", "15"),
		TableHeader:     cc("#9ccfd8", "116", "6"),
		TabsPrefix:      cc("#31748f", "67", "4"),
		TabsText:        cc("#e0def4", "189", "7"),
		TabsActiveText:  cc("#e0def4", "189", "15"),
		TabsActiveBg:    cc("#31748f", "67", "4"),
		SidebarTitle:    cc("#f6c177", "222", "3"),
		SidebarTitleHot: cc("#c4a7e7", "183", "5"),
	}
}

// Everforest Dark Medium theme.
func EverforestTheme() Theme {
	return Theme{
		Border:          cc("#4f585e", "239", "8"),
		BorderFocused:   cc("#a7c080", "142", "2"),
		DividerBorder:   cc("#4f585e", "239", "8"),
		OverlayBorder:   cc("#a7c080", "142", "2"),
		OverlayFooter:   cc("#e69875", "208", "3"),
		TextMuted:       cc("#7a8478", "243", "8"),
		TextPrimary:     cc("#d3c6aa", "223", "7"),
		TextAccent:      cc("#a7c080", "142", "2"),
		TextWarning:     cc("#dbbc7f", "214", "3"),
		TextError:       cc("#e67e80", "167", "1"),
		TextHeader:      cc("#d3c6aa", "223", "15"),
		TableHeader:     cc("#83c092", "108", "6"),
		TabsPrefix:      cc("#7fbbb3", "109", "6"),
		TabsText:        cc("#d3c6aa", "223", "7"),
		TabsActiveText:  cc("#d3c6aa", "223", "15"),
		TabsActiveBg:    cc("#7fbbb3", "109", "4"),
		SidebarTitle:    cc("#dbbc7f", "214", "3"),
		SidebarTitleHot: cc("#a7c080", "142", "2"),
	}
}

// Catppuccin Mocha theme.
func CatppuccinTheme() Theme {
	return Theme{
		Border:          cc("#6c7086", "60", "8"),
		BorderFocused:   cc("#cba6f7", "183", "5"),
		DividerBorder:   cc("#6c7086", "60", "8"),
		OverlayBorder:   cc("#cba6f7", "183", "5"),
		OverlayFooter:   cc("#fab387", "216", "3"),
		TextMuted:       cc("#a6adc8", "146", "8"),
		TextPrimary:     cc("#cdd6f4", "189", "7"),
		TextAccent:      cc("#cba6f7", "183", "5"),
		TextWarning:     cc("#f9e2af", "223", "3"),
		TextError:       cc("#f38ba8", "211", "1"),
		TextHeader:      cc("#cdd6f4", "231", "15"),
		TableHeader:     cc("#b4befe", "147", "5"),
		TabsPrefix:      cc("#94e2d5", "116", "6"),
		TabsText:        cc("#cdd6f4", "189", "7"),
		TabsActiveText:  cc("#cdd6f4", "231", "15"),
		TabsActiveBg:    cc("#cba6f7", "183", "5"),
		SidebarTitle:    cc("#f5c2e7", "225", "15"),
		SidebarTitleHot: cc("#cba6f7", "183", "5"),
	}
}

// Gruvbox Dark theme.
func GruvboxTheme() Theme {
	return Theme{
		Border:          cc("#7c6f64", "241", "8"),
		BorderFocused:   cc("#83a598", "109", "6"),
		DividerBorder:   cc("#7c6f64", "241", "8"),
		OverlayBorder:   cc("#83a598", "109", "6"),
		OverlayFooter:   cc("#fe8019", "173", "3"),
		TextMuted:       cc("#928374", "102", "8"),
		TextPrimary:     cc("#ebdbb2", "223", "7"),
		TextAccent:      cc("#83a598", "109", "6"),
		TextWarning:     cc("#fabd2f", "214", "3"),
		TextError:       cc("#fb4934", "203", "1"),
		TextHeader:      cc("#fbf1c7", "229", "15"),
		TableHeader:     cc("#8ec07c", "108", "2"),
		TabsPrefix:      cc("#d79921", "136", "3"),
		TabsText:        cc("#ebdbb2", "223", "7"),
		TabsActiveText:  cc("#fbf1c7", "229", "15"),
		TabsActiveBg:    cc("#458588", "66", "4"),
		SidebarTitle:    cc("#fabd2f", "222", "3"),
		SidebarTitleHot: cc("#8ec07c", "108", "2"),
	}
}

// One Dark theme.
func OneDarkTheme() Theme {
	return Theme{
		Border:          cc("#4b5263", "59", "8"),
		BorderFocused:   cc("#61afef", "75", "4"),
		DividerBorder:   cc("#4b5263", "59", "8"),
		OverlayBorder:   cc("#61afef", "75", "4"),
		OverlayFooter:   cc("#e5c07b", "222", "3"),
		TextMuted:       cc("#5c6370", "103", "8"),
		TextPrimary:     cc("#abb2bf", "145", "7"),
		TextAccent:      cc("#61afef", "75", "4"),
		TextWarning:     cc("#e5c07b", "222", "3"),
		TextError:       cc("#e06c75", "168", "1"),
		TextHeader:      cc("#ffffff", "231", "15"),
		TableHeader:     cc("#c678dd", "134", "5"),
		TabsPrefix:      cc("#56b6c2", "73", "6"),
		TabsText:        cc("#abb2bf", "145", "7"),
		TabsActiveText:  cc("#ffffff", "231", "15"),
		TabsActiveBg:    cc("#61afef", "75", "4"),
		SidebarTitle:    cc("#e5c07b", "222", "3"),
		SidebarTitleHot: cc("#98c379", "107", "2"),
	}
}

// SetNamedTheme sets the process-wide theme by name and returns that theme.
// The second return is false if the name is unknown; then the theme is unchanged and
// the returned theme is the current one (caller can ignore it or keep using Current).
func SetNamedTheme(name string) (Theme, bool) {
	var th Theme
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default", "stoat":
		th = DefaultTheme()
	case "catppuccin":
		th = CatppuccinTheme()
	case "dracula":
		th = DraculaTheme()
	case "everforest":
		th = EverforestTheme()
	case "rose-pine":
		th = RosePineTheme()
	case "princess":
		th = PrincessTheme()
	case "one-shell":
		th = OneShellTheme()
	case "blueish":
		th = BlueishTheme()
	case "gruvbox":
		th = GruvboxTheme()
	case "one-dark":
		th = OneDarkTheme()
	case "solarized":
		th = SolarizedTheme()
	default:
		return Current, false
	}
	Current = th
	return th, true
}
