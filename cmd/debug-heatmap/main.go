package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rgrouls/terminal-wrapped/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/debug-heatmap/main.go <path_to_zsh_history>")
		return
	}

	histPath := os.Args[1]
	fmt.Printf("Analyzing %s...\n", histPath)

	commands, err := parser.ParseZshHistory(histPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Parsed %d commands.\n", len(commands))

	// Manual Heatmap Analysis to debug
	dirCounts := make(map[string]int)
	zCounts := make(map[string]int)

	for _, cmd := range commands {
		parts := strings.Fields(cmd.Command)
		if len(parts) > 1 {
			baseCmd := parts[0]
			if baseCmd == "cd" {
				// Extract full arg
				idx := strings.Index(cmd.Command, " ")
				if idx != -1 {
					dir := strings.TrimSpace(cmd.Command[idx:])
					dir = cleanPath(dir)
					if dir != "" && dir != ".." && dir != "." && dir != "-" {
						dirCounts[dir]++
					}
				}
			} else if baseCmd == "z" {
				arg := parts[1] // z usually takes one main arg for jump?
				zCounts[arg]++
			}
		}
	}

	fmt.Println("\n--- Top 'cd' Destinations ---")
	printTop(dirCounts, 20)

	fmt.Println("\n--- Top 'z' Queries ---")
	printTop(zCounts, 20)
}

func printTop(counts map[string]int, n int) {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range counts {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	for i, kv := range ss {
		if i >= n {
			break
		}
		fmt.Printf("%d. '%s': %d\n", i+1, kv.Key, kv.Value)
	}
}

// Copy of cleanPath from analyzer
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

	// Normalize home
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.Split(path, "/")
		if len(parts) > 2 {
			path = "~/" + strings.Join(parts[3:], "/")
		}
	}

	return path
}
