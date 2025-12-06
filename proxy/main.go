package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	NebiusBaseURL = "https://api.tokenfactory.nebius.com/v1/chat/completions"
	ModelName     = "openai/gpt-oss-120b"
	StatsFile     = "stats.json"
)

// CommandCount represents a command or alias and its frequency
type CommandCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Value string `json:"value,omitempty"`
}

// RoastRequest is the data sent by the CLI
type RoastRequest struct {
	TotalCommands      int            `json:"total_commands"`
	UniqueCommands     int            `json:"unique_commands"`
	TopCommands        []CommandCount `json:"top_commands"`
	TopAliases         []CommandCount `json:"top_aliases"`
	MostActiveHour     int            `json:"most_active_hour"`
	MostActiveDay      string         `json:"most_active_day"`
	MostActiveMonth    string         `json:"most_active_month"`
	LongestStreak      int            `json:"longest_streak"`
	LongestStreakStart string         `json:"longest_streak_start"`
	LongestStreakEnd   string         `json:"longest_streak_end"`
}

// NebiusRequest is the payload sent to Nebius AI
type NebiusRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type NebiusResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// StatsStore handles simple analytics
type StatsStore struct {
	mu    sync.Mutex
	Stats []UserStat `json:"stats"`
}

type UserStat struct {
	TotalCommands int       `json:"total_commands"`
	LongestStreak int       `json:"longest_streak"`
	Timestamp     time.Time `json:"timestamp"`
}

var store = &StatsStore{}

func (s *StatsStore) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(StatsFile)
	if err != nil {
		return // File might not exist yet
	}
	json.Unmarshal(data, &s.Stats)
}

func (s *StatsStore) Save() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := json.Marshal(s.Stats)
	os.WriteFile(StatsFile, data, 0644)
}

func (s *StatsStore) Add(stat UserStat) {
	s.mu.Lock()
	s.Stats = append(s.Stats, stat)
	s.mu.Unlock()
	s.Save()
}

func (s *StatsStore) GetPercentiles(current UserStat) (float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Stats) == 0 {
		return 100.0, 100.0
	}

	cmdCount := 0
	streakCount := 0

	for _, stat := range s.Stats {
		if stat.TotalCommands < current.TotalCommands {
			cmdCount++
		}
		if stat.LongestStreak < current.LongestStreak {
			streakCount++
		}
	}

	cmdPercentile := float64(cmdCount) / float64(len(s.Stats)) * 100
	streakPercentile := float64(streakCount) / float64(len(s.Stats)) * 100

	return cmdPercentile, streakPercentile
}

func main() {
	apiKey := strings.TrimSpace(os.Getenv("NEBIUS_API_KEY"))
	if apiKey == "" {
		log.Fatal("NEBIUS_API_KEY environment variable is not set")
	}

	store.Load()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/roast", handleRoast)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Proxy server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRoast(w http.ResponseWriter, r *http.Request) {
	// ... (CORS handling remains same)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RoastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Construct prompt based on mode
	var systemPrompt, userPrompt string

	if req.Mode == "intro" {
		systemPrompt = `You are a witty, slightly snarky CLI assistant. 
Your goal is to write a short, engaging intro for a "Spotify Wrapped" style summary of the user's terminal usage.
Return ONLY valid JSON. No markdown formatting.`

		userPrompt = fmt.Sprintf(`
Analyze this user's terminal stats:
- Total Commands: %d
- Unique Commands: %d
- Most Active Hour: %d:00
- Most Active Month: %s

Return a JSON object with these keys:
- "intro": A short, funny welcome message (2-3 sentences). Mention their total commands.
- "rhythm": A comment on their peak activity time (hour/month).

Example JSON:
{
  "intro": "Welcome back, digital wizard! You smashed 10,000 commands this year.",
  "rhythm": "You love coding at 2 AM. Do you even sleep?"
}`, req.TotalCommands, req.UniqueCommands, req.MostActiveHour, req.MostActiveMonth)

	} else {
		// Details mode (default)

		// Record stats
		currentStat := UserStat{
			TotalCommands: req.TotalCommands,
			LongestStreak: req.LongestStreak,
			Timestamp:     time.Now(),
		}
		store.Add(currentStat)
		cmdPercentile, streakPercentile := store.GetPercentiles(currentStat)

		// Check for early adopters
		earlyAdopterMsg := ""
		store.mu.Lock()
		statsCount := len(store.Stats)
		store.mu.Unlock()

		// ... (Percentile logic)
		if statsCount < 4 {
			earlyAdopterMsg = " (You are one of the first 4 users! Come back later for global stats.)"
		} else {
			earlyAdopterMsg = fmt.Sprintf(" (Top %.1f%% of users!)", cmdPercentile)
		}

		streakMsg := ""
		if statsCount < 4 {
			streakMsg = " (You are one of the first 4 users! Come back later for global stats.)"
		} else {
			streakMsg = fmt.Sprintf(" (Top %.1f%% of users!)", streakPercentile)
		}

		streakInfo := fmt.Sprintf("%d days (from %s to %s)", req.LongestStreak, req.LongestStreakStart, req.LongestStreakEnd)
		if req.LongestStreak == 0 {
			streakInfo = "No streak yet"
		}

		topCmds := ""
		for _, cmd := range req.TopCommands {
			topCmds += fmt.Sprintf("- %s: %d\n", cmd.Name, cmd.Count)
		}

		topAliases := ""
		for _, cmd := range req.TopAliases {
			topAliases += fmt.Sprintf("- %s: %d\n", cmd.Name, cmd.Count)
		}

		systemPrompt = `You are a witty, slightly snarky CLI assistant.
Your goal is to generate specific remarks and stories for a "Spotify Wrapped" style summary.
Return ONLY valid JSON. No markdown formatting.`

		userPrompt = fmt.Sprintf(`
Analyze these detailed stats:
- Top Commands:
%s
- Top Aliases:
%s
- Longest Streak: %s %s
- Most Active Day: %s
- Most Active Hour: %d:00

Return a JSON object with these keys:
- "top_command_remarks": A JSON object mapping each top command name to a very short (3-5 words) witty remark.
- "alias_remarks": A JSON object mapping each top alias name to a very short (3-5 words) witty remark.
			Messages: []Message{
				{Role: "system", Content: "You are a helpful assistant that outputs JSON."},
				{Role: "user", Content: prompt},
			},
			Temperature:    0.7,
			ResponseFormat: &ResponseFormat{Type: "json_object"}, // Force JSON mode if supported, or just prompt engineering
		}

		jsonData, err := json.Marshal(nebiusReq)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		apiReq, err := http.NewRequest("POST", NebiusBaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		apiReq.Header.Set("Content-Type", "application/json")
		apiReq.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(apiReq)
		if err != nil {
			log.Printf("Error calling Nebius: %v", err)
			http.Error(w, fmt.Sprintf("Failed to contact AI provider: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Nebius API error: %s", string(body))
			http.Error(w, fmt.Sprintf("AI provider returned error: %d %s", resp.StatusCode, string(body)), http.StatusBadGateway)
			return
		}

		var nebiusResp NebiusResponse
		if err := json.NewDecoder(resp.Body).Decode(&nebiusResp); err != nil {
			http.Error(w, "Failed to decode AI response", http.StatusInternalServerError)
			return
		}

		if len(nebiusResp.Choices) == 0 {
			http.Error(w, "No response from AI", http.StatusBadGateway)
			return
		}

		// Return just the text
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(nebiusResp.Choices[0].Message.Content))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Proxy server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
