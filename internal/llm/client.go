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
	Mode               string                  `json:"mode"`
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
	// Extended Analysis
	TopDirectories  []analyzer.CommandCount `json:"top_directories"`
	ComplexityScore float64                 `json:"complexity_score"`
	TopEditors      []analyzer.CommandCount `json:"top_editors"`
}

// GenerateIntro fetches the intro story (fast)
func (c *Client) GenerateIntro(stats *analyzer.Analysis) (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	reqBody := RoastRequest{
		Mode:            "intro",
		TotalCommands:   stats.TotalCommands,
		UniqueCommands:  stats.UniqueCommands,
		MostActiveHour:  stats.MostActiveHour,
		MostActiveMonth: stats.MostActiveMonth,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	logRequest("intro", jsonData)

	resp, err := c.client.Post(c.proxyURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logError("intro", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logStatusError("intro", resp.StatusCode)
		return nil, fmt.Errorf("proxy returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logResponse("intro", body)

	cleanBody := cleanMarkdown(string(body))
	var storyMap map[string]string
	if err := json.Unmarshal([]byte(cleanBody), &storyMap); err != nil {
		return map[string]string{"intro": cleanBody}, nil
	}

	return storyMap, nil
}

// GenerateDetails fetches the detailed remarks (slow)
func (c *Client) GenerateDetails(stats *analyzer.Analysis) (map[string]string, map[string]string, map[string]string, error) {
	if c == nil {
		return nil, nil, nil, fmt.Errorf("client not initialized")
	}

	reqBody := RoastRequest{
		Mode:               "details",
		TotalCommands:      stats.TotalCommands,
		UniqueCommands:     stats.UniqueCommands,
		MostActiveHour:     stats.MostActiveHour,
		MostActiveDay:      stats.MostActiveDay,
		MostActiveMonth:    stats.MostActiveMonth,
		LongestStreak:      stats.LongestStreak,
		LongestStreakStart: stats.LongestStreakStart.Format("Jan 02"),
		LongestStreakEnd:   stats.LongestStreakEnd.Format("Jan 02"),
		ComplexityScore:    stats.ComplexityScore,
	}

	for i, cmd := range stats.TopCommands {
		if i >= 5 {
			break
		}
		reqBody.TopCommands = append(reqBody.TopCommands, cmd)
	}

	for i, cmd := range stats.TopAliases {
		if i >= 5 {
			break
		}
		reqBody.TopAliases = append(reqBody.TopAliases, cmd)
	}

	for i, cmd := range stats.TopDirectories {
		if i >= 5 {
			break
		}
		reqBody.TopDirectories = append(reqBody.TopDirectories, cmd)
	}

	for i, cmd := range stats.TopEditors {
		if i >= 5 {
			break
		}
		reqBody.TopEditors = append(reqBody.TopEditors, cmd)
	}

	jsonData, err := json.Marshal(reqBody)
	// ...
	if err != nil {
		return nil, nil, nil, err
	}

	logRequest("details", jsonData)

	resp, err := c.client.Post(c.proxyURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logError("details", err)
		return nil, nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logStatusError("details", resp.StatusCode)
		return nil, nil, nil, fmt.Errorf("proxy returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, nil, err
	}
	logResponse("details", body)

	cleanBody := cleanMarkdown(string(body))
	var storyMap map[string]interface{}
	if err := json.Unmarshal([]byte(cleanBody), &storyMap); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse details JSON: %v", err)
	}

	stories := make(map[string]string)
	for k, v := range storyMap {
		if str, ok := v.(string); ok {
			stories[k] = str
		}
	}

	cmdRemarks := make(map[string]string)
	if remarks, ok := storyMap["top_command_remarks"].(map[string]interface{}); ok {
		for k, v := range remarks {
			if str, ok := v.(string); ok {
				cmdRemarks[k] = str
			}
		}
	}

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

// Helpers for logging and cleaning
func logRequest(mode string, data []byte) {
	f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(fmt.Sprintf("--- %s Request at %s ---\n%s\n\n", mode, time.Now().Format(time.RFC3339), string(data)))
}

func logError(mode string, err error) {
	f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(fmt.Sprintf("--- %s Error at %s ---\n%v\n\n", mode, time.Now().Format(time.RFC3339), err))
}

func logStatusError(mode string, status int) {
	f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(fmt.Sprintf("--- %s Status Error at %s ---\nStatus: %d\n\n", mode, time.Now().Format(time.RFC3339), status))
}

func logResponse(mode string, body []byte) {
	f, _ := os.OpenFile("llm_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(fmt.Sprintf("--- %s Response at %s ---\n%s\n\n", mode, time.Now().Format(time.RFC3339), string(body)))
}

func cleanMarkdown(body string) string {
	cleanBody := strings.TrimSpace(body)
	if strings.HasPrefix(cleanBody, "```json") {
		cleanBody = strings.TrimPrefix(cleanBody, "```json")
		cleanBody = strings.TrimSuffix(cleanBody, "```")
	} else if strings.HasPrefix(cleanBody, "```") {
		cleanBody = strings.TrimPrefix(cleanBody, "```")
		cleanBody = strings.TrimSuffix(cleanBody, "```")
	}
	return strings.TrimSpace(cleanBody)
}
