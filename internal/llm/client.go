package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rgrouls/terminal-wrapped/internal/analyzer"
)

const (
	// Default to the deployed proxy
	DefaultProxyURL = "http://145.38.190.202:8080/roast"
)

type Client struct {
	proxyURL string
	client   *http.Client
}

func NewClient() *Client {
	proxyURL := os.Getenv("TERMINAL_WRAPPED_PROXY")
	if proxyURL == "" {
		proxyURL = DefaultProxyURL
	}

	return &Client{
		proxyURL: proxyURL,
		client:   &http.Client{Timeout: 90 * time.Second},
	}
}

type RoastRequest struct {
	TotalCommands      int                     `json:"total_commands"`
	UniqueCommands     int                     `json:"unique_commands"`
	TopCommands        []analyzer.CommandCount `json:"top_commands"`
	TopAliases         []analyzer.CommandCount `json:"top_aliases"`
	MostActiveHour     int                     `json:"most_active_hour"`
	MostActiveDay      string                  `json:"most_active_day"`
	MostActiveMonth    string                  `json:"most_active_month"`
	LongestStreak      int                     `json:"longest_streak"`
	LongestStreakStart string                  `json:"longest_streak_start"`
	LongestStreakEnd   string                  `json:"longest_streak_end"`
}

// GenerateStory fetches a structured story from the proxy
func (c *Client) GenerateStory(stats *analyzer.Analysis) (map[string]string, map[string]string, map[string]string, error) {
	if c == nil {
		return nil, nil, nil, fmt.Errorf("client not initialized")
	}

	// Construct request from stats
	reqBody := RoastRequest{
		TotalCommands:      stats.TotalCommands,
		UniqueCommands:     stats.UniqueCommands,
		MostActiveHour:     stats.MostActiveHour,
		MostActiveDay:      stats.MostActiveDay,
		MostActiveMonth:    stats.MostActiveMonth,
		LongestStreak:      stats.LongestStreak,
		LongestStreakStart: stats.LongestStreakStart.Format("Jan 02"),
		LongestStreakEnd:   stats.LongestStreakEnd.Format("Jan 02"),
	}

	// Get top 5 commands
	for i, cmd := range stats.TopCommands {
		if i >= 5 {
			break
		}
		reqBody.TopCommands = append(reqBody.TopCommands, cmd)
	}

	// Get top 5 aliases
	for i, cmd := range stats.TopAliases {
		if i >= 5 {
			break
		}
		reqBody.TopAliases = append(reqBody.TopAliases, cmd)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, nil, err
	}

	// Debug logging - Request
	f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(fmt.Sprintf("--- Request at %s ---\n%s\n\n", time.Now().Format(time.RFC3339), string(jsonData)))
	f.Close()

	resp, err := c.client.Post(c.proxyURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Debug logging - Error
		f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		f.WriteString(fmt.Sprintf("--- Error at %s ---\n%v\n\n", time.Now().Format(time.RFC3339), err))
		f.Close()
		return nil, nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Debug logging - Status Error
		f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		f.WriteString(fmt.Sprintf("--- Status Error at %s ---\nStatus: %d\n\n", time.Now().Format(time.RFC3339), resp.StatusCode))
		f.Close()
		return nil, nil, nil, fmt.Errorf("proxy returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	// Debug logging
	f, _ = os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(fmt.Sprintf("--- Response at %s ---\n%s\n\n", time.Now().Format(time.RFC3339), string(body)))

	// Clean up markdown code blocks if present
	cleanBody := string(body)
	cleanBody = strings.TrimSpace(cleanBody)
	if strings.HasPrefix(cleanBody, "```json") {
		cleanBody = strings.TrimPrefix(cleanBody, "```json")
		cleanBody = strings.TrimSuffix(cleanBody, "```")
	} else if strings.HasPrefix(cleanBody, "```") {
		cleanBody = strings.TrimPrefix(cleanBody, "```")
		cleanBody = strings.TrimSuffix(cleanBody, "```")
	}
	cleanBody = strings.TrimSpace(cleanBody)

	// Parse the JSON response from the proxy (which is the raw LLM output)
	var storyMap map[string]interface{}
	if err := json.Unmarshal([]byte(cleanBody), &storyMap); err != nil {
		// Fallback: try to return the raw text as intro if JSON fails
		// This helps debug if the LLM returns plain text
		return map[string]string{"intro": cleanBody}, nil, nil, nil
	}

	// Extract string stories
	stories := make(map[string]string)
	for k, v := range storyMap {
		if str, ok := v.(string); ok {
			stories[k] = str
		}
	}

	// Extract command remarks
	cmdRemarks := make(map[string]string)
	if remarks, ok := storyMap["top_command_remarks"].(map[string]interface{}); ok {
		for k, v := range remarks {
			if str, ok := v.(string); ok {
				cmdRemarks[k] = str
			}
		}
	}

	// Extract alias remarks
	aliasRemarks := make(map[string]string)
	if remarks, ok := storyMap["alias_remarks"].(map[string]interface{}); ok {
		for k, v := range remarks {
			if str, ok := v.(string); ok {
				aliasRemarks[k] = str
			}
		}
	}

	return stories, cmdRemarks, aliasRemarks, nil
}
