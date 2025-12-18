package analyzer

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rgrouls/terminal-wrapped/internal/parser"
)

type Analysis struct {
	TotalCommands      int
	UniqueCommands     int
	TopCommands        []CommandCount
	TopAliases         []CommandCount
	HourlyStats        map[int]int
	DailyStats         map[string]int
	MonthlyStats       map[string]int
	MostActiveHour     int
	MostActiveDay      string
	MostActiveMonth    string
	MaxDaily           int
	MaxMonthly         int
	MaxMonthlyAdjusted int // For visualization (95th percentile)

	// New Stats
	Punchcard          map[time.Weekday]map[int]int // Day -> Hour -> Count
	DayOfWeekStats     map[time.Weekday]int
	CurrentStreak      int
	LongestStreak      int
	LongestStreakStart time.Time
	LongestStreakEnd   time.Time

	// Extended Analysis
	TopDirectories  []CommandCount
	ComplexityScore float64
	PipeCount       int
	RedirectCount   int
	ChainCount      int
	TopEditors      []CommandCount
}

type CommandCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Value string `json:"value,omitempty"`
}

// Analyze processes the commands and returns statistics
func Analyze(commands []parser.Command, aliases map[string]string) *Analysis {
	stats := &Analysis{
		TotalCommands:  len(commands),
		HourlyStats:    make(map[int]int),
		DailyStats:     make(map[string]int),
		MonthlyStats:   make(map[string]int),
		Punchcard:      make(map[time.Weekday]map[int]int),
		DayOfWeekStats: make(map[time.Weekday]int),
	}

	// Initialize Punchcard
	for d := time.Sunday; d <= time.Saturday; d++ {
		stats.Punchcard[d] = make(map[int]int)
	}

	commandCounts := make(map[string]int)
	aliasCounts := make(map[string]int)
	dirCounts := make(map[string]int)
	editorCounts := make(map[string]int)
	uniqueCmds := make(map[string]bool)

	// For streak calculation
	activeDays := make(map[string]bool)
	var sortedDays []string

	editors := map[string]bool{
		"vim": true, "nvim": true, "vi": true,
		"nano": true, "pico": true,
		"emacs": true,
		"code":  true, "subl": true, "atom": true,
		"cursor": true,
	}

	totalComplexity := 0

	for _, cmd := range commands {
		uniqueCmds[cmd.Command] = true

		// Hourly stats
		stats.HourlyStats[cmd.Timestamp.Hour()]++

		// Daily stats
		dayKey := cmd.Timestamp.Format("2006-01-02")
		stats.DailyStats[dayKey]++
		activeDays[dayKey] = true

		// Monthly stats
		monthKey := cmd.Timestamp.Format("2006-01")
		stats.MonthlyStats[monthKey]++

		// Punchcard & DayOfWeek
		weekday := cmd.Timestamp.Weekday()
		stats.Punchcard[weekday][cmd.Timestamp.Hour()]++
		stats.DayOfWeekStats[weekday]++

		// Complexity Score
		complexity := 0
		pipes := strings.Count(cmd.Command, "|")
		redirects := strings.Count(cmd.Command, ">")
		chains := strings.Count(cmd.Command, "&&") + strings.Count(cmd.Command, ";")

		complexity += pipes
		complexity += redirects
		complexity += chains

		stats.PipeCount += pipes
		stats.RedirectCount += redirects
		stats.ChainCount += chains

		totalComplexity += complexity

		// Command counts
		parts := strings.Fields(cmd.Command)
		if len(parts) > 0 {
			baseCmd := parts[0]

			// Resolve alias for directory analysis purpose
			// (We still count the alias name itself in commandCounts below)
			expandedCmd := baseCmd
			expandedParts := parts
			if val, ok := aliases[baseCmd]; ok {
				expandedCmd = val
				// Re-tokenize the expanded command
				expandedParts = strings.Fields(expandedCmd)
				// Append original arguments if any
				if len(parts) > 1 {
					expandedParts = append(expandedParts, parts[1:]...)
				}
			}

			// Directory Heatmap (cd / z)
			// Check against the EXPANDED command
			cmdToCheck := baseCmd
			if len(expandedParts) > 0 {
				cmdToCheck = expandedParts[0]
			}

			if cmdToCheck == "cd" || cmdToCheck == "z" {
				if len(expandedParts) > 1 {
					// Join all parts after command to handle spaces
					rawTarget := strings.Join(expandedParts[1:], " ")

					// 1. Basic string cleaning
					dir := cleanPath(rawTarget)

					// 2. Normalize
					dir = filepath.Clean(dir)

					// 3. Segment and Count
					// We split by standard separator
					// cleanPath might leave us with things like "~/code/projects"
					// We want to count "code", "projects", "surfcontroller" separately.

					// 3. Hierarchical Counting
					// Instead of splitting by separator, we walk up the tree.
					// e.g. "~/code/projects" -> count "~/code/projects", count "~/code"

					currentPath := dir
					currentUser := os.Getenv("USER")

					// Avoid infinite loops just in case
					for i := 0; i < 20; i++ {
						if currentPath == "" || currentPath == "." || currentPath == "/" || currentPath == "~" {
							break
						}

						// Noise filtering
						// Ignore /Users and /Users/username
						if currentPath == "Users" || currentPath == "/Users" {
							goto NextParent
						}
						if currentUser != "" {
							if currentPath == "Users/"+currentUser || currentPath == "/Users/"+currentUser {
								goto NextParent
							}
							if currentPath == currentUser { // relative case
								goto NextParent
							}
						}

						// Count this path
						dirCounts[currentPath]++

					NextParent:
						next := filepath.Dir(currentPath)
						if next == currentPath { // Root reached
							break
						}
						currentPath = next
					}
				}
			}

			// Check for aliases (original logic)
			if _, isAlias := aliases[baseCmd]; isAlias {
				aliasCounts[baseCmd]++
			} else {
				commandCounts[baseCmd]++
			}

			// Editor Wars
			if editors[baseCmd] {
				editorCounts[baseCmd]++
			}
		}
	}

	if len(commands) > 0 {
		stats.ComplexityScore = float64(totalComplexity) / float64(len(commands))
	}

	stats.UniqueCommands = len(uniqueCmds)

	// Process Top Commands
	for name, count := range commandCounts {
		stats.TopCommands = append(stats.TopCommands, CommandCount{Name: name, Count: count})
	}
	sort.Slice(stats.TopCommands, func(i, j int) bool {
		return stats.TopCommands[i].Count > stats.TopCommands[j].Count
	})
	if len(stats.TopCommands) > 10 {
		stats.TopCommands = stats.TopCommands[:10]
	}

	// Process Top Aliases
	for name, count := range aliasCounts {
		val := aliases[name]
		stats.TopAliases = append(stats.TopAliases, CommandCount{Name: name, Count: count, Value: val})
	}
	sort.Slice(stats.TopAliases, func(i, j int) bool {
		return stats.TopAliases[i].Count > stats.TopAliases[j].Count
	})
	if len(stats.TopAliases) > 10 {
		stats.TopAliases = stats.TopAliases[:10]
	}

	// Process Top Directories
	for name, count := range dirCounts {
		stats.TopDirectories = append(stats.TopDirectories, CommandCount{Name: name, Count: count})
	}
	sort.Slice(stats.TopDirectories, func(i, j int) bool {
		return stats.TopDirectories[i].Count > stats.TopDirectories[j].Count
	})
	if len(stats.TopDirectories) > 5 {
		stats.TopDirectories = stats.TopDirectories[:5]
	}

	// Process Top Editors
	for name, count := range editorCounts {
		stats.TopEditors = append(stats.TopEditors, CommandCount{Name: name, Count: count})
	}
	sort.Slice(stats.TopEditors, func(i, j int) bool {
		return stats.TopEditors[i].Count > stats.TopEditors[j].Count
	})
	if len(stats.TopEditors) > 5 {
		stats.TopEditors = stats.TopEditors[:5]
	}

	// Find Max values
	for _, count := range stats.DailyStats {
		if count > stats.MaxDaily {
			stats.MaxDaily = count
		}
	}

	// Monthly stats with outlier handling
	var monthlyCounts []int
	for m, count := range stats.MonthlyStats {
		if count > stats.MaxMonthly {
			stats.MaxMonthly = count
			stats.MostActiveMonth = m
		}
		monthlyCounts = append(monthlyCounts, count)
	}

	// Calculate 95th percentile for visualization to handle anomalies
	if len(monthlyCounts) > 0 {
		sort.Ints(monthlyCounts)
		idx := int(math.Ceil(0.95*float64(len(monthlyCounts)))) - 1
		if idx < 0 {
			idx = 0
		}
		stats.MaxMonthlyAdjusted = monthlyCounts[idx]
		// If the max is significantly larger (e.g. 2x) than 95th percentile, use 95th
		if stats.MaxMonthly < int(float64(stats.MaxMonthlyAdjusted)*1.5) {
			stats.MaxMonthlyAdjusted = stats.MaxMonthly
		}
	}

	// Find most active hour
	maxHourCount := 0
	for h, count := range stats.HourlyStats {
		if count > maxHourCount {
			maxHourCount = count
			stats.MostActiveHour = h
		}
	}

	// Find most active day
	maxDayCount := 0
	for d, count := range stats.DayOfWeekStats {
		if count > maxDayCount {
			maxDayCount = count
			stats.MostActiveDay = d.String()
		}
	}

	// Calculate Streaks
	for day := range activeDays {
		sortedDays = append(sortedDays, day)
	}
	sort.Strings(sortedDays)

	currentStreak := 0
	longestStreak := 0

	if len(sortedDays) > 0 {
		// Calculate longest streak
		tempStreak := 1
		tempStart, _ := time.Parse("2006-01-02", sortedDays[0])

		for i := 1; i < len(sortedDays); i++ {
			prev, _ := time.Parse("2006-01-02", sortedDays[i-1])
			curr, _ := time.Parse("2006-01-02", sortedDays[i])

			if curr.Sub(prev).Hours() <= 26 { // Allow some leeway for "next day"
				tempStreak++
			} else {
				if tempStreak > longestStreak {
					longestStreak = tempStreak
					stats.LongestStreakStart = tempStart
					stats.LongestStreakEnd = prev
				}
				tempStreak = 1
				tempStart = curr
			}
		}
		if tempStreak > longestStreak {
			longestStreak = tempStreak
			stats.LongestStreakStart = tempStart
			// For the last item, it is the end
			lastDay, _ := time.Parse("2006-01-02", sortedDays[len(sortedDays)-1])
			stats.LongestStreakEnd = lastDay
		}
		stats.LongestStreak = longestStreak

		// Calculate current streak
		// Check if the last active day was today or yesterday
		lastActive, _ := time.Parse("2006-01-02", sortedDays[len(sortedDays)-1])
		now := time.Now()
		diff := now.Sub(lastActive).Hours()

		if diff < 48 { // If active within last 48 hours, count streak
			// Backtrack from end
			currentStreak = 1
			for i := len(sortedDays) - 1; i > 0; i-- {
				curr, _ := time.Parse("2006-01-02", sortedDays[i])
				prev, _ := time.Parse("2006-01-02", sortedDays[i-1])
				if curr.Sub(prev).Hours() <= 26 {
					currentStreak++
				} else {
					break
				}
			}
			stats.CurrentStreak = currentStreak
		} else {
			stats.CurrentStreak = 0
		}
	}

	return stats
}

// ParseAliases reads the .zshrc file to find aliases
func ParseAliases(homeDir string) (map[string]string, error) {
	aliases := make(map[string]string)
	file, err := os.Open(homeDir + "/.zshrc")
	if err != nil {
		// If .zshrc doesn't exist, just return empty map, don't error out
		return aliases, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "alias ") {
			// Format: alias name='command'
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				cmd := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases[name] = cmd
			}
		}
	}
	return aliases, nil
}

func cleanPath(path string) string {
	// Remove surrounding quotes
	if len(path) >= 2 {
		if (path[0] == '"' && path[len(path)-1] == '"') ||
			(path[0] == '\'' && path[len(path)-1] == '\'') {
			path = path[1 : len(path)-1]
		}
	}

	// Handle escaped spaces
	path = strings.ReplaceAll(path, "\\ ", " ")

	// Remove trailing slash if present (but not if it's just "/")
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}
