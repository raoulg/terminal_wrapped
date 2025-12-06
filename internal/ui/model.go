package ui

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rgrouls/terminal-wrapped/internal/analyzer"
	"github.com/rgrouls/terminal-wrapped/internal/llm"
)

type Model struct {
	Analysis           *analyzer.Analysis
	CurrentSlide       int
	Slides             []Slide
	Quitting           bool
	AIStories          map[string]string // Map of slide title -> story
	TopCommandRemarks  map[string]string
	AliasRemarks       map[string]string
	LoadingAI          bool
	AIResponseReceived bool
	AnimationPercent   float64 // 0.0 to 1.0
	Progress           progress.Model
	LoadingMsgIndex    int
	LastMsgUpdate      time.Time
}

type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type Slide interface {
	Title() string
	Content(m Model) string
}

type AIStoryMsg struct {
	Stories           map[string]string
	TopCommandRemarks map[string]string
	AliasRemarks      map[string]string
}

var loadingMessages = []string{
	"🤖 AI is analyzing your terminal history...",
	"🤔 Hmmm, interesting command choices...",
	"🔥 Calculating your longest streak...",
	"👀 Judging your alias usage...",
	"⏳ Almost there...",
	"😅 Just a second...",
	"🚀 Finalizing your roast...",
}

func NewModel(analysis *analyzer.Analysis) Model {
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return Model{
		Analysis: analysis,
		Slides: []Slide{
			IntroSlide{},
			RhythmSlide{},
			StreakSlide{}, // New
			TopCommandsSlide{},
			AliasSlide{},
			PunchcardSlide{}, // New
			HourlySlide{},
			MonthlySlide{},
			OutroSlide{},
		},
		LoadingAI:          true,
		AIResponseReceived: false,
		AIStories:          make(map[string]string),
		TopCommandRemarks:  make(map[string]string),
		AliasRemarks:       make(map[string]string),
		AnimationPercent:   0.0,
		Progress:           prog,
		LoadingMsgIndex:    0,
		LastMsgUpdate:      time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			client := llm.NewClient()
			if client == nil {
				return AIStoryMsg{Stories: nil}
			}

			// Fetch structured story
			stories, cmdRemarks, aliasRemarks, err := client.GenerateStory(m.Analysis)
			if err != nil {
				return AIStoryMsg{Stories: nil}
			}
			return AIStoryMsg{
				Stories:           stories,
				TopCommandRemarks: cmdRemarks,
				AliasRemarks:      aliasRemarks,
			}
		},
		tick(), // Start animation
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AIStoryMsg:
		m.AIStories = msg.Stories
		m.TopCommandRemarks = msg.TopCommandRemarks
		m.AliasRemarks = msg.AliasRemarks
		m.AIResponseReceived = true
		// Don't set LoadingAI = false yet, let the progress bar finish
	case TickMsg:
		// Animate progress bar if loading
		if m.LoadingAI {
			percent := m.Progress.Percent()

			// Increment logic
			increment := 0.0

			if m.AIResponseReceived {
				// Finish fast!
				increment = 0.05
			} else {
				if percent < 0.95 {
					// Fast(er) phase: 0 to 95% in about 15 seconds
					// 15 seconds / 0.05s tick = 300 ticks
					// 0.95 / 300 = ~0.0032
					increment = 0.0035
				} else if percent < 0.99 {
					// Stalling phase: very slow
					increment = 0.0001
				}
			}

			if percent+increment < 1.0 {
				cmd := m.Progress.SetPercent(percent + increment)

				// Cycle messages every 3 seconds
				if time.Since(m.LastMsgUpdate) > 3*time.Second {
					m.LoadingMsgIndex = (m.LoadingMsgIndex + 1) % len(loadingMessages)
					m.LastMsgUpdate = time.Now()
				}

				return m, tea.Batch(cmd, tick())
			} else {
				// Done!
				m.LoadingAI = false
				return m, tick()
			}
		}

		// Animate charts if not loading
		if m.AnimationPercent < 1.0 {
			m.AnimationPercent += 0.05
			if m.AnimationPercent > 1.0 {
				m.AnimationPercent = 1.0
			}
			return m, tick()
		}
	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "s":
			if m.AIResponseReceived {
				htmlFile, err := GenerateHTML(m.Analysis, m.AIStories, m.TopCommandRemarks, m.AliasRemarks)
				if err == nil {
					// Open the file in the browser
					// This is a quick hack for macOS, ideally use a library
					_ = exec.Command("open", htmlFile).Start()
				}
			}
		case "n", " ", "right":
			if !m.LoadingAI && m.CurrentSlide < len(m.Slides)-1 {
				m.CurrentSlide++
				m.AnimationPercent = 0.0 // Reset animation
				return m, tick()
			}
		case "p", "left":
			if !m.LoadingAI && m.CurrentSlide > 0 {
				m.CurrentSlide--
				m.AnimationPercent = 0.0 // Reset animation
				return m, tick()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return "Bye! 👋\n"
	}

	// Show loading screen with progress bar if AI is loading
	if m.LoadingAI {
		msg := loadingMessages[m.LoadingMsgIndex]
		// If nearly full, force one of the "waiting" messages if not already showing one
		if m.Progress.Percent() > 0.95 && m.LoadingMsgIndex < 4 {
			m.LoadingMsgIndex = 4 // Jump to "Almost there..."
			msg = loadingMessages[m.LoadingMsgIndex]
		}

		return fmt.Sprintf(
			"\n\n   %s\n\n   %s\n\n",
			HelpStyle.Render(msg),
			m.Progress.View(),
		)
	}

	slide := m.Slides[m.CurrentSlide]

	// Header
	header := TitleStyle.Render("Terminal Wrapped 2024 🎁")

	// AI Story Section (Per slide)
	aiSection := ""

	// Check if there is a story for this slide
	// We map slide titles to story keys roughly
	key := ""
	title := m.Slides[m.CurrentSlide].Title()
	if strings.HasPrefix(title, "Terminal Wrapped") {
		key = "intro"
	}
	switch title {
	case "Terminal Rhythm":
		key = "rhythm"
	case "Top Commands":
		key = "top_commands"
	case "Alias Artistry":
		key = "aliases"
	case "Punchcard":
		key = "punchcard"
	case "Coding Streaks":
		key = "streaks"
	case "Daily Activity":
		key = "hourly"
	case "Monthly Activity":
		key = "monthly"
	}

	if story, ok := m.AIStories[key]; ok && story != "" {
		// Wrap text to 60 chars
		wrappedStory := lipgloss.NewStyle().Width(60).Render(story)
		aiSection = lipgloss.NewStyle().Foreground(ColorAccent).Italic(true).Render(fmt.Sprintf("🤖 %s", wrappedStory))
	}

	// Content
	content := slide.Content(m)

	// Footer with simple animation
	arrow := "→"
	if m.CurrentSlide == len(m.Slides)-1 {
		arrow = "↺"
	}

	footer := HelpStyle.Render(fmt.Sprintf("Slide %d/%d • n: next %s • p: prev • s: share • q: quit", m.CurrentSlide+1, len(m.Slides), arrow))

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", header, aiSection, content, footer)
}

// --- Slides ---

type IntroSlide struct{}

func (s IntroSlide) Title() string { return fmt.Sprintf("Terminal Wrapped %d 🎁", time.Now().Year()) }
func (s IntroSlide) Content(m Model) string {
	// Colorful ASCII Art Title
	title := `
  _______                  _             _ 
 |__   __|                (_)           | |
    | | ___ _ __ _ __ ___  _ _ __   __ _| |
    | |/ _ \ '__| '_ ' _ \| | '_ \ / _' | |
    | |  __/ |  | | | | | | | | | | (_| | |
    |_|\___|_|  |_| |_| |_|_|_| |_|\__,_|_|
                                           
 __          __                               _ 
 \ \        / /                              | |
  \ \  /\  / / __ __ _ _ __  _ __   ___  __| |
   \ \/  \/ / '__/ _' | '_ \| '_ \ / _ \/ _' |
    \  /\  | | | (_| | |_) | |_) |  __/ (_| |
     \/  \/ |_|  \__,_| .__/| .__/ \___|\__,_|
                      | |   | |               
                      |_|   |_|               
`
	gradTitle := ""
	lines := strings.Split(title, "\n")
	for i, line := range lines {
		intensity := float64(i) / float64(len(lines))
		gradTitle += lipgloss.NewStyle().Foreground(GetGradientColor(intensity)).Render(line) + "\n"
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "User"
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"Welcome, %s! 👋\n\n"+
			"We've analyzed %d commands from your history.\n"+
			"Let's see what you've been up to...",
		gradTitle,
		lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(user),
		m.Analysis.TotalCommands,
	)
}

type RhythmSlide struct{}

func (s RhythmSlide) Title() string { return "Terminal Rhythm" }
func (s RhythmSlide) Content(m Model) string {
	return fmt.Sprintf(
		"Total Commands: %d\n"+
			"Unique Commands: %d\n\n"+
			"Most Active Hour: %02d:00\n"+
			"Most Active Month: %s",
		m.Analysis.TotalCommands,
		m.Analysis.UniqueCommands,
		m.Analysis.MostActiveHour,
		m.Analysis.MostActiveMonth,
	)
}

type StreakSlide struct{}

func (s StreakSlide) Title() string { return "Coding Streaks" }
func (s StreakSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Consistency is key! 🔑") + "\n\n")

	sb.WriteString(fmt.Sprintf("🔥 Current Streak: %d days\n", m.Analysis.CurrentStreak))
	sb.WriteString(fmt.Sprintf("🏆 Longest Streak: %d days\n", m.Analysis.LongestStreak))

	if m.Analysis.LongestStreak > 0 {
		start := m.Analysis.LongestStreakStart.Format("Jan 02")
		end := m.Analysis.LongestStreakEnd.Format("Jan 02")
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("   (%s - %s)", start, end)) + "\n")
	}

	if m.Analysis.CurrentStreak > 7 {
		sb.WriteString("\nWow! You're on fire! Keep it up! 🚀")
	} else if m.Analysis.CurrentStreak == 0 {
		sb.WriteString("\nNo active streak... Time to start one today! 💪")
	}

	return sb.String()
}

type PunchcardSlide struct{}

func (s PunchcardSlide) Title() string { return "Punchcard" }

// Update PunchcardSlide to use AnimationPercent
func (s PunchcardSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Your coding hotspots (Hour vs Day):") + "\n\n")

	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	// Header row (Hours)
	sb.WriteString("    ")
	for h := 0; h < 24; h += 3 {
		sb.WriteString(fmt.Sprintf("%02d ", h))
	}
	sb.WriteString("\n")

	maxCount := 0
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			if count := m.Analysis.Punchcard[time.Weekday(d)][h]; count > maxCount {
				maxCount = count
			}
		}
	}

	for d, dayName := range days {
		sb.WriteString(fmt.Sprintf("%s ", dayName))
		for h := 0; h < 24; h++ {
			count := m.Analysis.Punchcard[time.Weekday(d)][h]
			char := "·"
			color := ColorMuted

			if count > 0 {
				intensity := LogScale(count, maxCount)

				// Animate intensity
				if intensity > m.AnimationPercent {
					intensity = m.AnimationPercent // Clip to current animation progress
				}

				if intensity > 0.75 {
					char = "█"
				} else if intensity > 0.5 {
					char = "▓"
				} else if intensity > 0.25 {
					char = "▒"
				} else {
					char = "░"
				}

				color = GetGradientColor(intensity)
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(color).Render(char))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type TopCommandsSlide struct{}

func (s TopCommandsSlide) Title() string { return "Top Commands" }

// Update TopCommandsSlide
func (s TopCommandsSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Your most used commands (excluding aliases):") + "\n\n")

	maxCount := 0
	if len(m.Analysis.TopCommands) > 0 {
		maxCount = m.Analysis.TopCommands[0].Count
	}

	for _, cmd := range m.Analysis.TopCommands {
		barWidth := 30
		// Animate bar width
		targetFilled := LogScale(cmd.Count, maxCount) * float64(barWidth)
		filled := int(targetFilled * m.AnimationPercent)

		intensity := LogScale(cmd.Count, maxCount)
		color := GetGradientColor(intensity)

		bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
			BarBackgroundStyle.Render(strings.Repeat("░", barWidth-filled))

		remark := ""
		if r, ok := m.TopCommandRemarks[cmd.Name]; ok {
			remark = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render(" " + r)
		} else if r, ok := CommandRemarks[cmd.Name]; ok {
			// Fallback to hardcoded
			remark = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render(" " + r)
		}

		sb.WriteString(fmt.Sprintf("%-10s %s %d%s\n", cmd.Name, bar, cmd.Count, remark))
	}
	return sb.String()
}

type AliasSlide struct{}

func (s AliasSlide) Title() string { return "Alias Artistry" }

// Update AliasSlide
func (s AliasSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Your favorite aliases:") + "\n\n")

	if len(m.Analysis.TopAliases) == 0 {
		return "No aliases found! Try adding some to your .zshrc to save time."
	}

	maxCount := m.Analysis.TopAliases[0].Count

	for _, cmd := range m.Analysis.TopAliases {
		barWidth := 30
		targetFilled := LogScale(cmd.Count, maxCount) * float64(barWidth)
		filled := int(targetFilled * m.AnimationPercent)

		intensity := LogScale(cmd.Count, maxCount)
		color := GetGradientColor(intensity)

		bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
			BarBackgroundStyle.Render(strings.Repeat("░", barWidth-filled))

		remark := ""
		if r, ok := m.AliasRemarks[cmd.Name]; ok {
			remark = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render(" " + r)
		}

		sb.WriteString(fmt.Sprintf("%-15s %s %d%s\n", cmd.Name, bar, cmd.Count, remark))
	}
	return sb.String()
}

type HourlySlide struct{}

func (s HourlySlide) Title() string { return "Daily Activity" }

// Update HourlySlide
func (s HourlySlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("When you are most active:") + "\n\n")

	maxCount := 0
	for _, count := range m.Analysis.HourlyStats {
		if count > maxCount {
			maxCount = count
		}
	}

	for h := 0; h < 24; h++ {
		count := m.Analysis.HourlyStats[h]
		if count > 0 {
			barWidth := 50
			targetFilled := LogScale(count, maxCount) * float64(barWidth)
			filled := int(targetFilled * m.AnimationPercent)

			intensity := LogScale(count, maxCount)
			color := GetGradientColor(intensity)

			bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
			sb.WriteString(fmt.Sprintf("%02d:00 %s %d\n", h, bar, count))
		}
	}
	return sb.String()
}

type MonthlySlide struct{}

func (s MonthlySlide) Title() string { return "Monthly Activity" }

// Update MonthlySlide
func (s MonthlySlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Your year in review:") + "\n\n")

	var months []string
	for m := range m.Analysis.MonthlyStats {
		months = append(months, m)
	}
	sort.Strings(months)

	maxCount := m.Analysis.MaxMonthlyAdjusted
	if maxCount == 0 {
		maxCount = 1
	}

	for _, month := range months {
		count := m.Analysis.MonthlyStats[month]
		visualCount := count
		if visualCount > maxCount {
			visualCount = maxCount
		}

		barWidth := 50
		targetFilled := LogScale(visualCount, maxCount) * float64(barWidth)
		filled := int(targetFilled * m.AnimationPercent)

		barChar := "█"
		color := GetGradientColor(LogScale(visualCount, maxCount))

		if count > maxCount {
			barChar = "▓"
			color = ColorSecondary // Highlight outlier
		}

		bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat(barChar, filled))

		sb.WriteString(fmt.Sprintf("%s %s %d\n", month, bar, count))
	}
	return sb.String()
}

type OutroSlide struct{}

func (s OutroSlide) Title() string { return "Goodbye" }
func (s OutroSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("That's a wrap! 🎬") + "\n\n")
	sb.WriteString("Thanks for using Terminal Wrapped.\n")
	sb.WriteString("Keep on coding and stay terminal-native! 🚀\n\n")

	// Add prominent Share CTA
	shareStyle := lipgloss.NewStyle().
		Foreground(ColorDark).
		Background(ColorAccent).
		Bold(true).
		Padding(1, 2).
		MarginTop(2)

	sb.WriteString(shareStyle.Render("PRESS 's' TO SHARE YOUR WRAPPED REPORT") + "\n\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("Press 'q' to quit."))
	return sb.String()
}
