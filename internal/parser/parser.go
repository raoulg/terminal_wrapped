package parser

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Command represents a parsed shell command with its timestamp
type Command struct {
	Command   string
	Timestamp time.Time
}

// ParseZshHistory parses a zsh history file and returns a list of Commands.
// It handles multi-line commands where lines end with a backslash.
func ParseZshHistory(filePath string) ([]Command, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var commands []Command
	var currentCommandBuilder strings.Builder
	var currentTimestamp time.Time
	var inCommand bool

	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle very long lines if necessary, 
    // but default 64k is usually fine for history. 
    // If lines are huge, we might need scanner.Buffer().

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is the start of a new command entry
		// Zsh history format: : <timestamp>:<duration>;<command>
		if strings.HasPrefix(line, ": ") {
			// If we were building a command, verify if it was actually finished
            // The previous line logic should have handled the append, but 
            // if we hit a new start marker, we force finish the previous one if it exists.
			if inCommand {
				commands = append(commands, Command{
					Command:   currentCommandBuilder.String(),
					Timestamp: currentTimestamp,
				})
				currentCommandBuilder.Reset()
				inCommand = false
			}

			// Parse metadata
			parts := strings.SplitN(line, ";", 2)
			if len(parts) < 2 {
				continue // Malformed line
			}

			meta := parts[0]
			cmdPart := parts[1]

			// Extract timestamp
			// meta is like ": 1678886400:0"
			metaParts := strings.Split(meta, ":")
			if len(metaParts) < 2 {
				continue
			}
            
            // metaParts[0] is empty string (before first :)
            // metaParts[1] is timestamp (space stripped)
            tsStr := strings.TrimSpace(metaParts[1])
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				continue
			}
			currentTimestamp = time.Unix(ts, 0)

			// Check if command continues
			if strings.HasSuffix(cmdPart, "\\") {
				// Remove the backslash and keep building
				currentCommandBuilder.WriteString(cmdPart[:len(cmdPart)-1])
				inCommand = true
			} else {
				// Single line command
				commands = append(commands, Command{
					Command:   cmdPart,
					Timestamp: currentTimestamp,
				})
			}
		} else {
			// This is a continuation line
			if inCommand {
				if strings.HasSuffix(line, "\\") {
					currentCommandBuilder.WriteString(line[:len(line)-1])
				} else {
					currentCommandBuilder.WriteString(line)
					// End of command
					commands = append(commands, Command{
						Command:   currentCommandBuilder.String(),
						Timestamp: currentTimestamp,
					})
					currentCommandBuilder.Reset()
					inCommand = false
				}
			}
            // If not inCommand, ignore this line (orphaned continuation or garbage)
		}
	}

	// Handle case where file ends while building a command
	if inCommand {
		commands = append(commands, Command{
			Command:   currentCommandBuilder.String(),
			Timestamp: currentTimestamp,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commands, nil
}
