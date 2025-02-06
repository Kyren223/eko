package colors

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/pkg/assert"
)

const Count = 32

var (
	DefaultLightGray          = lipgloss.Color("#939AA3")
	DefaultGray               = lipgloss.Color("#585858")
	DefaultDarkGray           = lipgloss.Color("#313244")
	DefaultLightBlue          = lipgloss.Color("#5874FF")
	DefaultBlue               = lipgloss.Color("#0029f5")
	DefaultDarkCyan           = lipgloss.Color("#007E8A")
	DefaultDarkerCyan         = lipgloss.Color("#005d66")
	DefaultDarkerMidnightBlue = lipgloss.Color("#0e0e16")
	DefaultDarkMidnightBlue   = lipgloss.Color("#181825")
	DefaultMidnightBlue       = lipgloss.Color("#1E1E2E")
	DefaultLightMidnightBlue  = lipgloss.Color("#3c3c5d")
	DefaultTurquoise          = lipgloss.Color("#54D7A9")
	DefaultRed                = lipgloss.Color("#F16265")
	DefaultMutedRed           = lipgloss.Color("#443737")
	DefaultDarkMutedRed       = lipgloss.Color("#402d2d")
	DefaultWhite              = lipgloss.Color("#FFFFFF")
	DefaultBlack              = lipgloss.Color("#000000")
	DefaultGreen              = lipgloss.Color("#46d46c")
	DefaultPurple             = lipgloss.Color("#BB91F0")
	DefaultDarkPurple         = lipgloss.Color("#87123d")
	DefaultMutedPurple        = lipgloss.Color("#403744")
	DefaultDarkMutedPurple    = lipgloss.Color("#382d40")
	DefaultOrange             = lipgloss.Color("#F5A670")
	DefaultGold               = lipgloss.Color("#FFBF00")
	DefaultMutedGold          = lipgloss.Color("#444037")
	DefaultDarkMutedGold      = lipgloss.Color("#40382d")

	DefaultBackground          = DefaultMidnightBlue
	DefaultBackgroundHighlight = DefaultLightMidnightBlue
	DefaultBackgroundDim       = DefaultDarkMidnightBlue
	DefaultBackgroundDimmer    = DefaultDarkerMidnightBlue
	DefaultError               = DefaultRed
	DefaultFocus               = DefaultLightBlue
)

var (
	LightGray          = DefaultLightGray
	Gray               = DefaultGray
	DarkGray           = DefaultDarkGray
	LightBlue          = DefaultLightBlue
	Blue               = DefaultBlue
	DarkCyan           = DefaultDarkCyan
	DarkerCyan         = DefaultDarkerCyan
	DarkerMidnightBlue = DefaultDarkerMidnightBlue
	DarkMidnightBlue   = DefaultDarkMidnightBlue
	MidnightBlue       = DefaultMidnightBlue
	LightMidnightBlue  = DefaultLightMidnightBlue
	Turquoise          = DefaultTurquoise
	Red                = DefaultRed
	MutedRed           = DefaultMutedRed
	DarkMutedRed       = DefaultDarkMutedRed
	White              = DefaultWhite
	Black              = DefaultBlack
	Green              = DefaultGreen
	Purple             = DefaultPurple
	DarkPurple         = DefaultDarkPurple
	MutedPurple        = DefaultMutedPurple
	DarkMutedPurple    = DefaultDarkMutedPurple
	Orange             = DefaultOrange
	Gold               = DefaultGold
	MutedGold          = DefaultMutedGold
	DarkMutedGold      = DefaultDarkMutedGold

	Background          = MidnightBlue
	BackgroundHighlight = LightMidnightBlue
	BackgroundDim       = DarkMidnightBlue
	BackgroundDimmer    = DarkerMidnightBlue
	Error               = Red
	Focus               = LightBlue
)

var colors = make([]lipgloss.Color, Count)

func Save() {
	i := 0
	colors[i] = LightGray
	i++
	colors[i] = Gray
	i++
	colors[i] = DarkGray
	i++
	colors[i] = LightBlue
	i++
	colors[i] = Blue
	i++
	colors[i] = DarkCyan
	i++
	colors[i] = DarkerCyan
	i++
	colors[i] = DarkerMidnightBlue
	i++
	colors[i] = DarkMidnightBlue
	i++
	colors[i] = MidnightBlue
	i++
	colors[i] = LightMidnightBlue
	i++
	colors[i] = Turquoise
	i++
	colors[i] = Red
	i++
	colors[i] = MutedRed
	i++
	colors[i] = DarkMutedRed
	i++
	colors[i] = White
	i++
	colors[i] = Black
	i++
	colors[i] = Green
	i++
	colors[i] = Purple
	i++
	colors[i] = DarkPurple
	i++
	colors[i] = MutedPurple
	i++
	colors[i] = DarkMutedPurple
	i++
	colors[i] = Orange
	i++
	colors[i] = Gold
	i++
	colors[i] = MutedGold
	i++
	colors[i] = DarkMutedGold
	i++

	colors[i] = Background
	i++
	colors[i] = BackgroundHighlight
	i++
	colors[i] = BackgroundDim
	i++
	colors[i] = BackgroundDimmer
	i++
	colors[i] = Error
	i++
	colors[i] = Focus
	i++
}

func Restore() {
	Load(colors)
}

func Load(colors []lipgloss.Color) {
	i := 0
	LightGray = colors[i]
	i++
	Gray = colors[i]
	i++
	DarkGray = colors[i]
	i++
	LightBlue = colors[i]
	i++
	Blue = colors[i]
	i++
	DarkCyan = colors[i]
	i++
	DarkerCyan = colors[i]
	i++
	DarkerMidnightBlue = colors[i]
	i++
	DarkMidnightBlue = colors[i]
	i++
	MidnightBlue = colors[i]
	i++
	LightMidnightBlue = colors[i]
	i++
	Turquoise = colors[i]
	i++
	Red = colors[i]
	i++
	MutedRed = colors[i]
	i++
	DarkMutedRed = colors[i]
	i++
	White = colors[i]
	i++
	Black = colors[i]
	i++
	Green = colors[i]
	i++
	Purple = colors[i]
	i++
	DarkPurple = colors[i]
	i++
	MutedPurple = colors[i]
	i++
	DarkMutedPurple = colors[i]
	i++
	Orange = colors[i]
	i++
	Gold = colors[i]
	i++
	MutedGold = colors[i]
	i++
	DarkMutedGold = colors[i]
	i++

	Background = colors[i]
	i++
	BackgroundHighlight = colors[i]
	i++
	BackgroundDim = colors[i]
	i++
	BackgroundDimmer = colors[i]
	i++
	Error = colors[i]
	i++
	Focus = colors[i]
	i++
}

func LoadStrings(s []string) {
	assert.Assert(len(s) == Count, "len of strings must match color count")
	for i, c := range s {
		if IsHex(c) {
			colors[i] = lipgloss.Color(c)
		}
	}
	Restore()
}

func Get() []lipgloss.Color {
	return slices.Clone(colors)
}

func ToHex(color lipgloss.Color) string {
	r, g, b, _ := color.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

func IsHex(color string) bool {
	re := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	return re.MatchString(color)
}
