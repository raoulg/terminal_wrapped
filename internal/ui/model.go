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
	LoadingDetails     bool // New flag for background loading
	AIResponseReceived bool
	AnimationPercent   float64 // 0.0 to 1.0
	Progress           progress.Model
	DetailsProgress    progress.Model // New progress bar for details
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

type AIIntroMsg struct {
	Stories map[string]string
}

type AIDetailsMsg struct {
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

var detailLoadingMessages = []string{
	"✋ Not so fast! The AI is still roasting you...",
	"🔥 Servers are melting, give it a second...",
	"🤖 Computing maximum snark...",
	"⚡️ Hold your horses, speed racer...",
}

func NewModel(analysis *analyzer.Analysis) Model {
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	detailsProg := progress.New(progress.WithDefaultGradient())
	detailsProg.Width = 40

	return Model{
		Analysis: analysis,
		Slides: []Slide{
			IntroSlide{},
			RhythmSlide{},
			StreakSlide{},
			TopCommandsSlide{},
			AliasSlide{},
			DirectorySlide{},  // New
			ComplexitySlide{}, // New
			EditorSlide{},     // New
			PunchcardSlide{},
			HourlySlide{},
			MonthlySlide{},
			OutroSlide{},
		},
		LoadingAI:          true,
		LoadingDetails:     true, // Start loading details
		AIResponseReceived: false,
		AIStories:          make(map[string]string),
		TopCommandRemarks:  make(map[string]string),
		AliasRemarks:       make(map[string]string),
		AnimationPercent:   0.0,
		Progress:           prog,
		DetailsProgress:    detailsProg,
		LoadingMsgIndex:    0,
		LastMsgUpdate:      time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			client := llm.NewClient()
			if client == nil {
				return AIIntroMsg{Stories: nil}
			}

			// Fetch intro story (fast)
			stories, err := client.GenerateIntro(m.Analysis)
			if err != nil {
				return AIIntroMsg{Stories: nil}
			}
			return AIIntroMsg{Stories: stories}
		},
		tick(), // Start animation
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AIIntroMsg:
		// Intro received, stop loading and show UI
		for k, v := range msg.Stories {
			m.AIStories[k] = v
		}
		m.AIResponseReceived = true
		m.LoadingAI = false // Allow interaction immediately

		// Trigger details fetch in background
		return m, func() tea.Msg {
			client := llm.NewClient()
			if client == nil {
				return AIDetailsMsg{}
			}
			stories, cmdRemarks, aliasRemarks, err := client.GenerateDetails(m.Analysis)
			if err != nil {
				return AIDetailsMsg{}
			}
			return AIDetailsMsg{
				Stories:           stories,
				TopCommandRemarks: cmdRemarks,
				AliasRemarks:      aliasRemarks,
			}
		}

	case AIDetailsMsg:
		// Details received, update model silently
		for k, v := range msg.Stories {
			m.AIStories[k] = v
		}
		m.TopCommandRemarks = msg.TopCommandRemarks
		m.AliasRemarks = msg.AliasRemarks
		m.LoadingDetails = false // Details loaded!
		return m, nil

	case TickMsg:
		var cmds []tea.Cmd

		// Animate Intro progress bar if loading
		if m.LoadingAI {
			percent := m.Progress.Percent()
			increment := 0.0

			if m.AIResponseReceived {
				increment = 0.05
			} else {
				if percent < 0.95 {
					increment = 0.0035
				} else if percent < 0.99 {
					increment = 0.0001
				}
			}

			if percent+increment < 1.0 {
				cmd := m.Progress.SetPercent(percent + increment)
				cmds = append(cmds, cmd)

				if time.Since(m.LastMsgUpdate) > 3*time.Second {
					m.LoadingMsgIndex = (m.LoadingMsgIndex + 1) % len(loadingMessages)
					m.LastMsgUpdate = time.Now()
				}
			} else {
				m.LoadingAI = false
			}
		}

		// Animate Details progress bar if loading
		if m.LoadingDetails {
			percent := m.DetailsProgress.Percent()
			increment := 0.0

			// We don't know when it finishes until AIDetailsMsg comes,
			// so we just guess a slower pace than Intro
			if percent < 0.90 {
				// 0 to 90% in about 30 seconds
				increment = 0.0015
			} else if percent < 0.99 {
				// Stalling phase
				increment = 0.00005
			}

			// If we received details (LoadingDetails set to false in AIDetailsMsg),
			// this block won't run, but we want to ensure it fills up?
			// Actually AIDetailsMsg sets LoadingDetails=false immediately.
			// So we don't need a "finish fast" logic here because the UI will just switch to content.

			if percent+increment < 1.0 {
				cmd := m.DetailsProgress.SetPercent(percent + increment)
				cmds = append(cmds, cmd)
			}
		}

		// Animate Details progress bar if loading
		if m.LoadingDetails {
			percent := m.DetailsProgress.Percent()
			increment := 0.0

			// We don't know when it finishes until AIDetailsMsg comes,
			// so we just guess a slower pace than Intro
			if percent < 0.90 {
				// 0 to 90% in about 30 seconds
				increment = 0.0015
			} else if percent < 0.99 {
				// Stalling phase
				increment = 0.00005
			}

			if percent+increment < 1.0 {
				cmd := m.DetailsProgress.SetPercent(percent + increment)
				cmds = append(cmds, cmd)
			}
		}

		// Animate charts if not loading
		if !m.LoadingAI && m.AnimationPercent < 1.0 {
			m.AnimationPercent += 0.05
			if m.AnimationPercent > 1.0 {
				m.AnimationPercent = 1.0
			}
		}

		cmds = append(cmds, tick())
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		newProg, cmd1 := m.Progress.Update(msg)
		m.Progress = newProg.(progress.Model)

		newDetailsProg, cmd2 := m.DetailsProgress.Update(msg)
		m.DetailsProgress = newDetailsProg.(progress.Model)

		return m, tea.Batch(cmd1, cmd2)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "s":
			if m.AIResponseReceived {
				htmlFile, err := GenerateHTML(m.Analysis, m.AIStories, m.TopCommandRemarks, m.AliasRemarks)
				if err == nil {
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

	// Show loading screen with progress bar if AI is loading (Intro)
	if m.LoadingAI {
		msg := loadingMessages[m.LoadingMsgIndex]
		if m.Progress.Percent() > 0.95 && m.LoadingMsgIndex < 4 {
			m.LoadingMsgIndex = 4
			msg = loadingMessages[m.LoadingMsgIndex]
		}

		return fmt.Sprintf(
			"\n\n   %s\n\n   %s\n\n",
			HelpStyle.Render(msg),
			m.Progress.View(),
		)
	}

	// Check if user is trying to view details before they are loaded
	// Intro (0) and Rhythm (1) are safe. Others need details.
	if m.LoadingDetails && m.CurrentSlide > 1 {
		// Pick a random funny message based on time to make it feel alive
		msgIndex := int(time.Now().Unix()) % len(detailLoadingMessages)
		return fmt.Sprintf(
			"\n\n   %s\n\n   %s\n\n",
			HelpStyle.Render(detailLoadingMessages[msgIndex]),
			m.DetailsProgress.View(),
		)
	}

	slide := m.Slides[m.CurrentSlide]

	// Header
	header := TitleStyle.Render("Terminal Wrapped 2024 🎁")

	// AI Story Section (Per slide)
	aiSection := ""

	// Check if there is a story for this slide
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
	case "Top Destinations":
		key = "directory_heatmap"
	case "Hacker Level":
		key = "complexity"
	case "Editor Wars":
		key = "editor_wars"
	}

	if story, ok := m.AIStories[key]; ok && story != "" {
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

	for i, cmd := range m.Analysis.TopCommands {
		// Staggered animation: reveal one by one
		// Total animation time is 1.0. We have N items.
		// Item i reveals from (i/N) to ((i+1)/N) + overlap?
		// Simpler: visible if AnimationPercent > (i / N)

		threshold := float64(i) / float64(len(m.Analysis.TopCommands))
		if m.AnimationPercent < threshold {
			continue // Not visible yet
		}

		barWidth := 30
		// Animate bar width: it grows from 0 to full *after* it appears
		// Local percent for this item
		localPercent := (m.AnimationPercent - threshold) * float64(len(m.Analysis.TopCommands))
		if localPercent > 1.0 {
			localPercent = 1.0
		}

		targetFilled := LogScale(cmd.Count, maxCount) * float64(barWidth)
		filled := int(targetFilled * localPercent)

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

	// ... (Previous slides)

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

// --- New Extended Analysis Slides ---

type DirectorySlide struct{}

func (s DirectorySlide) Title() string { return "Top Destinations" }
func (s DirectorySlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Where you spend your time:") + "\n\n")

	if len(m.Analysis.TopDirectories) == 0 {
		return "No directory usage found. Do you even `cd`?"
	}

	maxCount := m.Analysis.TopDirectories[0].Count
	for _, cmd := range m.Analysis.TopDirectories {
		barWidth := 30
		targetFilled := LogScale(cmd.Count, maxCount) * float64(barWidth)
		filled := int(targetFilled * m.AnimationPercent)

		intensity := LogScale(cmd.Count, maxCount)
		color := GetGradientColor(intensity)

		bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
			BarBackgroundStyle.Render(strings.Repeat("░", barWidth-filled))

		sb.WriteString(fmt.Sprintf("%-20s %s %d\n", cmd.Name, bar, cmd.Count))
	}
	return sb.String()
}

type ComplexitySlide struct{}

func (s ComplexitySlide) Title() string { return "Hacker Level" }
func (s ComplexitySlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Command Complexity Score:") + "\n\n")

	score := m.Analysis.ComplexityScore
	level := "Script Kiddie 👶"
	if score > 0.5 {
		level = "Junior Dev 👨‍💻"
	}
	if score > 1.0 {
		level = "Bash Wizard 🧙‍♂️"
	}
	if score > 2.0 {
		level = "10x Engineer 🚀"
	}

	sb.WriteString(fmt.Sprintf("Score: %.2f\n", score))
	sb.WriteString(fmt.Sprintf("Level: %s\n\n", lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(level)))

	sb.WriteString("Breakdown:\n")
	sb.WriteString(fmt.Sprintf("- Pipes (|): %d\n", m.Analysis.PipeCount))
	sb.WriteString(fmt.Sprintf("- Redirects (>): %d\n", m.Analysis.RedirectCount))
	sb.WriteString(fmt.Sprintf("- Chains (&&, ;): %d\n", m.Analysis.ChainCount))

	return sb.String()
}

type EditorSlide struct{}

func (s EditorSlide) Title() string { return "Editor Wars" }
func (s EditorSlide) Content(m Model) string {
	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render("Your weapon of choice:") + "\n\n")

	if len(m.Analysis.TopEditors) == 0 {
		return "No editor usage found. real programmers use `cat` > file.c?"
	}

	maxCount := m.Analysis.TopEditors[0].Count
	for _, cmd := range m.Analysis.TopEditors {
		barWidth := 30
		targetFilled := LogScale(cmd.Count, maxCount) * float64(barWidth)
		filled := int(targetFilled * m.AnimationPercent)

		intensity := LogScale(cmd.Count, maxCount)
		color := GetGradientColor(intensity)

		bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
			BarBackgroundStyle.Render(strings.Repeat("░", barWidth-filled))

		sb.WriteString(fmt.Sprintf("%-10s %s %d\n", cmd.Name, bar, cmd.Count))
	}
	return sb.String()
}
