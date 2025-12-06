package ui

import (
	"math"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#04B575") // Green
	ColorSecondary = lipgloss.Color("#FF4136") // Red
	ColorAccent    = lipgloss.Color("#FFDC00") // Yellow
	ColorMuted     = lipgloss.Color("#AAAAAA") // Grey
	ColorDark      = lipgloss.Color("#333333")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			MarginBottom(1)

	StatsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			MarginLeft(2)

	BarStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	BarBackgroundStyle = lipgloss.NewStyle().
				Foreground(ColorDark)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(2)

	// Gradients (simulated with colors for now, as Lipgloss gradients are for text)
	GradientColors = []lipgloss.Color{
		lipgloss.Color("#04B575"), // Green
		lipgloss.Color("#3CD070"),
		lipgloss.Color("#75EB6B"),
		lipgloss.Color("#B0FF66"), // Yellow-Green
	}
)

func GetGradientColor(intensity float64) lipgloss.Color {
	idx := int(intensity * float64(len(GradientColors)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(GradientColors) {
		idx = len(GradientColors) - 1
	}
	return GradientColors[idx]
}

// LogScale calculates the height of a bar using logarithmic scaling
// Returns a value between 0.0 and 1.0
func LogScale(count, maxCount int) float64 {
	if count <= 0 {
		return 0
	}
	if maxCount <= 1 {
		return 1
	}

	// log(count) / log(max)
	// We add 1 to avoid log(0) issues if count is 1, but count >= 1 here.
	// Actually, let's use log10
	val := math.Log10(float64(count))
	max := math.Log10(float64(maxCount))

	if max == 0 {
		return 0
	}
	return val / max
}

var CommandRemarks = map[string]string{
	"ls":     "Checking if files are still there? 👀",
	"cd":     "Exploring the filesystem universe 🚀",
	"git":    "Commit early, commit often! 💾",
	"docker": "It works on my machine! 🐳",
	"grep":   "Searching for a needle in a haystack 🧐",
	"cat":    "Meow? 🐱",
	"vim":    "Trapped in vim? :q! to escape! 🆘",
	"nvim":   "Ah, a person of culture 🎩",
	"python": "Indentation matters! 🐍",
	"go":     "Gopher it! 🐹",
	"cargo":  "Rustacean detected 🦀",
	"npm":    "Downloading the internet... 📦",
	"yarn":   "Kinda like npm, but faster? 🧶",
	"ssh":    "Knock knock, let me in! 🔑",
	"sudo":   "With great power comes great responsibility ⚡",
	"rm":     "Living dangerously! 🗑️",
	"curl":   "Fetching the web, one byte at a time 🌐",
	"htop":   "Watching those cores burn 🔥",
	"clear":  "A fresh start ✨",
}
